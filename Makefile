APP := nats-jetstream-producer
IMAGE ?= $(APP):latest

.PHONY: tidy build run docker fmt vet clean

tidy:        ## Resolve deps and write go.sum (run this first)
	go mod tidy

build: tidy  ## Build the binary into ./bin
	go build -trimpath -ldflags="-s -w" -o bin/$(APP) .

run:         ## Run locally (reads env / .env exported by your shell)
	go run .

fmt:
	go fmt ./...

vet:
	go vet ./...

docker:      ## Build the container image
	docker build -t $(IMAGE) .

clean:
	rm -rf bin dist
