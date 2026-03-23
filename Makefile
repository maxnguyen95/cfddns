APP_NAME := cfddns
IMAGE := maxnguyen95/cfddns:latest

.PHONY: help docker-once docker-run docker-logs docker-stop compose-up compose-logs compose-down test source-run-once source-run build

help:
	@printf "%s\n" \
		"docker-once     Run the published image once with .env" \
		"docker-run      Run the published image in the background" \
		"docker-logs     Follow container logs" \
		"docker-stop     Stop and remove the container" \
		"compose-up      Start the service with docker compose" \
		"compose-logs    Follow docker compose logs" \
		"compose-down    Stop docker compose" \
		"test            Run Go tests" \
		"source-run-once Run from source once using .env" \
		"source-run      Run from source continuously using .env" \
		"build           Build a local binary into ./bin"

docker-once:
	docker run --rm --env-file .env $(IMAGE) --once

docker-run:
	docker run -d --name $(APP_NAME) --restart unless-stopped --env-file .env $(IMAGE)

docker-logs:
	docker logs -f $(APP_NAME)

docker-stop:
	docker stop $(APP_NAME)
	docker rm $(APP_NAME)

compose-up:
	docker compose up -d

compose-logs:
	docker compose logs -f

compose-down:
	docker compose down

test:
	go test ./...

source-run-once:
	go run ./cmd/cfddns --once

source-run:
	go run ./cmd/cfddns

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -o bin/$(APP_NAME) ./cmd/cfddns
