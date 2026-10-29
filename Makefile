MODULE := github.com/hubtel/keda-otel-collector-fleet-scaler
IMG ?= ghcr.io/hubtel/keda-otel-collector-fleet-scaler:dev
GO ?= go

.PHONY: all generate build test fmt vet tidy docker-build

all: generate test build

generate:
	@./hack/generate-proto.sh

build:
	$(GO) build -o bin/scaler ./cmd/scaler

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

docker-build:
	docker build -t $(IMG) .
