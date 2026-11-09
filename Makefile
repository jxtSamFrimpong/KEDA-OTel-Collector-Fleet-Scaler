MODULE := github.com/jxtSamFrimpong/keda-otel-collector-fleet-scaler
IMG ?= keda-otel-collector-fleet-scaler:dev
GO ?= go

.PHONY: all generate build test fmt vet tidy docker-build

all: generate test build

generate:
	@./hack/generate-proto.sh

build:
	$(GO) build -o bin/scaler ./cmd/scaler

# Scaler packages only — deploy/demo/cmd/loadgen is a standalone demo (extra deps).
TEST_PKGS := ./pkg/... ./cmd/... ./api/...

test:
	$(GO) test $(TEST_PKGS)

fmt:
	$(GO) fmt $(TEST_PKGS)

vet:
	$(GO) vet $(TEST_PKGS)

tidy:
	$(GO) mod tidy

docker-build:
	docker build -t $(IMG) .
