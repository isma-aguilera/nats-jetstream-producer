package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// config holds all runtime settings. Everything is sourced from environment
// variables so that no connection string, credential or topology is baked into
// the binary. Sensible defaults make local development friction-free.
type config struct {
	NATSURL        string        // nats:// or tls:// URL of the server/cluster
	CredsFile      string        // optional path to a NATS .creds file for auth
	StreamName     string        // JetStream stream to publish into
	StreamSubjects []string      // subjects bound to the stream (only used if it must be created)
	StreamReplicas int           // replica count used only when creating the stream
	Subject        string        // subject each message is published to
	Interval       time.Duration // delay between publishes
	MaxMessages    int           // stop after N messages; 0 means run forever
}

func loadConfig() (config, error) {
	c := config{
		NATSURL:        getenv("NATS_URL", "nats://localhost:4222"),
		CredsFile:      os.Getenv("NATS_CREDS"),
		StreamName:     getenv("STREAM_NAME", "LAB"),
		StreamSubjects: []string{getenv("STREAM_SUBJECTS", "lab.>")},
		Subject:        getenv("SUBJECT", "lab.events"),
	}

	interval, err := time.ParseDuration(getenv("PUBLISH_INTERVAL", "1s"))
	if err != nil {
		return c, fmt.Errorf("invalid PUBLISH_INTERVAL: %w", err)
	}
	c.Interval = interval

	replicas, err := strconv.Atoi(getenv("STREAM_REPLICAS", "1"))
	if err != nil || replicas < 1 {
		return c, fmt.Errorf("invalid STREAM_REPLICAS %q: must be a positive integer", os.Getenv("STREAM_REPLICAS"))
	}
	c.StreamReplicas = replicas

	maxMsgs, err := strconv.Atoi(getenv("MAX_MESSAGES", "0"))
	if err != nil || maxMsgs < 0 {
		return c, fmt.Errorf("invalid MAX_MESSAGES %q: must be a non-negative integer", os.Getenv("MAX_MESSAGES"))
	}
	c.MaxMessages = maxMsgs

	return c, nil
}

// getenv returns the value of key, or def when the variable is unset or empty.
func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
