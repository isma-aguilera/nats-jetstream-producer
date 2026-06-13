# --- build stage ---
FROM golang:1.23-alpine AS build
WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
# Static, stripped binary so it runs on a scratch/distroless base.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/producer .

# --- runtime stage ---
# distroless static + nonroot: no shell, no package manager, runs as UID 65532.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/producer /producer
USER nonroot:nonroot
# No EXPOSE: the producer is a client and listens on no port.
ENTRYPOINT ["/producer"]
