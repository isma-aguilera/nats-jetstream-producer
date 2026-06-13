# nats-jetstream-producer

A small, dependency-light **Go** service that publishes JSON messages to a
[NATS JetStream](https://docs.nats.io/nats-concepts/jetstream) stream at a fixed
interval. Part of a NATS JetStream on Kubernetes lab, paired with
[`nats-jetstream-consumer`](https://github.com/isma-aguilera/nats-jetstream-consumer).

## Design notes

- **No secrets in code.** Connection URL and credentials come from environment
  variables; authentication uses a file-based `.creds` (mounted as a secret),
  never a hardcoded token.
- **No exposed ports.** This is a NATS *client*; it opens no listening socket.
- **Graceful shutdown** on `SIGINT`/`SIGTERM` (drains the connection).
- **Structured JSON logs** via `log/slog`.
- Ships as a **distroless, non-root** container.

## Configuration

| Variable           | Default                  | Description                                            |
| ------------------ | ------------------------ | ------------------------------------------------------ |
| `NATS_URL`         | `nats://localhost:4222`  | Server/cluster URL (`tls://` supported).               |
| `NATS_CREDS`       | _(unset)_                | Path to a NATS `.creds` file for authenticated setups. |
| `STREAM_NAME`      | `LAB`                    | Stream to publish into.                                |
| `STREAM_SUBJECTS`  | `lab.>`                  | Subjects bound to the stream (only if it's created).   |
| `STREAM_REPLICAS`  | `1`                      | Replicas used only when creating the stream.           |
| `SUBJECT`          | `lab.events`             | Subject each message is published to.                  |
| `PUBLISH_INTERVAL` | `1s`                     | Delay between publishes (Go duration).                 |
| `MAX_MESSAGES`     | `0`                      | Stop after N messages; `0` = forever.                  |

## Run locally

```bash
make tidy          # first time: resolve deps + create go.sum

# Reach the in-cluster NATS from your machine:
kubectl -n nats-system port-forward svc/nats 4222:4222 &

export NATS_URL=nats://localhost:4222
make run
```

## Container

```bash
make docker
docker run --rm -e NATS_URL=nats://host.docker.internal:4222 nats-jetstream-producer:latest
```

## License

MIT (or your choice).
