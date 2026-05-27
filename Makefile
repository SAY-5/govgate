GO_DIR := apps/register
PY_DIR := apps/reporter

.PHONY: all test test-go test-py test-integration lint lint-go lint-py typecheck \
        build build-go build-images run-go bench bench-regress fmt tidy

all: lint typecheck test

## --- Go ---

build-go:
	cd $(GO_DIR) && go build -o bin/govgate ./cmd/govgate

run-go: build-go
	cd $(GO_DIR) && ./bin/govgate serve

test-go:
	cd $(GO_DIR) && go test ./...

test-integration:
	cd $(GO_DIR) && go test -tags=integration ./...

lint-go:
	cd $(GO_DIR) && go vet ./...

tidy:
	cd $(GO_DIR) && go mod tidy

bench:
	cd $(GO_DIR) && go test -run=^$$ -bench=. -benchmem ./internal/scoring/... ./internal/register/...

bench-regress:
	cd $(GO_DIR) && go run ./cmd/govgate benchregress --threshold 0.30

## --- Python ---

PY := cd $(PY_DIR) && poetry run

lint-py:
	$(PY) ruff check src tests

fmt:
	$(PY) ruff format src tests
	cd $(GO_DIR) && gofmt -w .

test-py:
	$(PY) pytest --cov --cov-report=term-missing

typecheck-py:
	$(PY) mypy

## --- Aggregate ---

lint: lint-go lint-py

typecheck: lint-go typecheck-py

test: test-go test-py

build: build-go

build-images:
	docker build -t govgate-register $(GO_DIR)
	docker build -t govgate-reporter $(PY_DIR)
