// Command nats-jetstream-producer publishes JSON messages to a NATS JetStream
// stream at a fixed interval. It is a pure client: it opens no listening port
// and reads every setting (including credentials) from the environment.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(logger); err != nil {
		logger.Error("producer terminated with error", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Cancel the context on SIGINT/SIGTERM so we shut down gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	nc, err := connect(cfg, logger)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	// Drain flushes pending messages and closes the connection cleanly.
	defer func() { _ = nc.Drain() }()

	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("create JetStream context: %w", err)
	}

	if err := ensureStream(ctx, js, cfg); err != nil {
		return fmt.Errorf("ensure stream %q: %w", cfg.StreamName, err)
	}

	logger.Info("producer started",
		"url", nc.ConnectedUrl(),
		"stream", cfg.StreamName,
		"subject", cfg.Subject,
		"interval", cfg.Interval.String(),
		"max_messages", cfg.MaxMessages,
	)

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	var sent int
	for {
		select {
		case <-ctx.Done():
			logger.Info("shutdown signal received", "messages_sent", sent)
			return nil

		case t := <-ticker.C:
			payload, err := json.Marshal(message{
				Sequence:  sent + 1,
				Timestamp: t.UTC().Format(time.RFC3339Nano),
				Producer:  "nats-jetstream-producer",
			})
			if err != nil {
				logger.Error("marshal message", "err", err)
				continue
			}

			// Bound each publish so a stalled server can't block shutdown.
			pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			ack, err := js.Publish(pubCtx, cfg.Subject, payload)
			cancel()
			if err != nil {
				logger.Error("publish failed", "subject", cfg.Subject, "err", err)
				continue
			}

			sent++
			logger.Info("published",
				"subject", cfg.Subject,
				"stream", ack.Stream,
				"stream_seq", ack.Sequence,
			)

			if cfg.MaxMessages > 0 && sent >= cfg.MaxMessages {
				logger.Info("reached MAX_MESSAGES, stopping", "messages_sent", sent)
				return nil
			}
		}
	}
}

// message is the JSON envelope written to each subject.
type message struct {
	Sequence  int    `json:"sequence"`
	Timestamp string `json:"timestamp"`
	Producer  string `json:"producer"`
}

// connect dials NATS with automatic reconnection and optional file-based
// credentials. Credentials are never embedded; they are loaded from the path
// given in NATS_CREDS, if any.
func connect(cfg config, logger *slog.Logger) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Name("nats-jetstream-producer"),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2 * time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			logger.Warn("disconnected from NATS", "err", err)
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			logger.Info("reconnected to NATS", "url", c.ConnectedUrl())
		}),
	}
	if cfg.CredsFile != "" {
		opts = append(opts, nats.UserCredentials(cfg.CredsFile))
	}
	return nats.Connect(cfg.NATSURL, opts...)
}

// ensureStream creates the stream if it does not already exist. If it exists
// (e.g. provisioned out-of-band) it is left untouched so we never clobber an
// operator-managed configuration.
func ensureStream(ctx context.Context, js jetstream.JetStream, cfg config) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := js.Stream(ctx, cfg.StreamName); err == nil {
		return nil
	} else if !errors.Is(err, jetstream.ErrStreamNotFound) {
		return err
	}

	_, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:      cfg.StreamName,
		Subjects:  cfg.StreamSubjects,
		Storage:   jetstream.FileStorage,
		Replicas:  cfg.StreamReplicas,
		Retention: jetstream.LimitsPolicy,
	})
	return err
}
