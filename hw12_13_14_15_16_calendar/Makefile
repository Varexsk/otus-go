BIN_DIR := ./bin
BIN := $(BIN_DIR)/calendar
BIN_SCHEDULER := $(BIN_DIR)/calendar_scheduler
BIN_SENDER := $(BIN_DIR)/calendar_sender
IMG_CALENDAR := calendar:develop
IMG_SCHEDULER := calendar-scheduler:develop
IMG_SENDER := calendar-sender:develop

GIT_HASH := $(shell git log --format="%h" -n 1)
LDFLAGS := -X main.release="develop" -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%S) -X main.gitHash=$(GIT_HASH)

#Postrges
POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= password
POSTGRES_DB ?= backend
POSTGRES_PORT ?= 5435
POSTGRES_CONTAINER := postgres-calendar

#Rabbit
RABBIT_USER ?= rabbit
RABBIT_PASSWORD ?= password
RABBIT_PORT ?= 5672
RABBIT_PORT_UI ?= 15672
RABBIT_CONTAINER := rabbitmq

DSN ?= postgres://calendar:calendar@localhost:5432/calendar?sslmode=disable

run-postgres:
	docker run -d --name $(POSTGRES_CONTAINER) \
	-e POSTGRES_USER=$(POSTGRES_USER) \
	-e POSTGRES_PASSWORD=$(POSTGRES_PASSWORD) \
	-e POSTGRES_DB=$(POSTGRES_DB) \
	-p $(POSTGRES_PORT):5432 \
	-v postgres-data:/var/lib/postgresql/data \
	postgres:latest

run-rabbit:
	docker run -d --name $(RABBIT_CONTAINER) \
	-p $(RABBIT_PORT):5672 \
	-p $(RABBIT_PORT_UI):15672 \
	-e RABBITMQ_DEFAULT_USER=$(RABBIT_USER) \
	-e RABBITMQ_DEFAULT_PASS=$(RABBIT_PASSWORD) \
	rabbitmq:management

build:
	go build -v -o $(BIN) -ldflags "$(LDFLAGS)" ./cmd/calendar
	go build -v -o $(BIN_SCHEDULER) -ldflags "$(LDFLAGS)" ./cmd/calendar_scheduler
	go build -v -o $(BIN_SENDER) -ldflags "$(LDFLAGS)" ./cmd/calendar_sender

run: build
	$(BIN) -config ./configs/calendar_config.yaml

run-scheduler: build
	$(BIN_SCHEDULER) -config ./configs/scheduler_config.yaml

run-sender: build
	$(BIN_SENDER) -config ./configs/sender_config.yaml

build-img:
	docker build --build-arg=LDFLAGS="$(LDFLAGS)" -t $(IMG_CALENDAR) -f build/calendar/Dockerfile .
	docker build --build-arg=LDFLAGS="$(LDFLAGS)" -t $(IMG_SCHEDULER) -f build/scheduler/Dockerfile .
	docker build --build-arg=LDFLAGS="$(LDFLAGS)" -t $(IMG_SENDER) -f build/sender/Dockerfile .

run-img: build-img
	docker run --rm -p 8888:8080 -e STORAGE_TYPE=memory $(IMG_CALENDAR)

COMPOSE := docker compose -f deployments/docker-compose.yaml

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

version: build
	$(BIN) version

test:
	go test -race ./...

install-lint-deps:
	(which golangci-lint > /dev/null) || go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.63.4

lint: install-lint-deps
	golangci-lint run ./...

migrate:
	goose -dir migrations postgres "$(DSN)" up

test-sql:
	CALENDAR_TEST_DSN="$(DSN)" go test -race -count=1 ./internal/storage/sql/...

RMQ_URL ?= amqp://$(RABBIT_USER):$(RABBIT_PASSWORD)@localhost:$(RABBIT_PORT)/

test-rmq:
	CALENDAR_TEST_RMQ_URL="$(RMQ_URL)" go test -race -count=1 ./internal/rmq/...

generate:
	go generate ./...

.PHONY: build run run-scheduler run-sender build-img run-img up down version test lint migrate test-sql test-rmq generate

