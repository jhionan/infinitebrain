.PHONY: all build test test-integration lint fmt vet clean \
        db-migrate db-rollback db-diff db-generate \
        proto mockery docker-up docker-down security-scan vuln-check run-local help

# ─── Variables ────────────────────────────────────────────────────────────────
BINARY_NAME     := infinite-brain
BINARY_PATH     := bin/$(BINARY_NAME)
MAIN_PKG        := ./cmd/server
GO_FILES        := $(shell find . -name '*.go' -not -path './vendor/*' -not -path './db/sqlc/*')
COVERAGE_OUT    := coverage.out
COVERAGE_HTML   := coverage.html
COVERAGE_MIN    := 90

# ─── Default ──────────────────────────────────────────────────────────────────
all: fmt vet lint test build

# ─── Build ────────────────────────────────────────────────────────────────────
build:
	@echo "→ Building $(BINARY_NAME)..."
	@mkdir -p bin
	go build -ldflags="-s -w" -o $(BINARY_PATH) $(MAIN_PKG)
	@echo "✓ Built $(BINARY_PATH)"

run: build
	@./$(BINARY_PATH)

run-local:
	@echo "→ Running with local dev config..."
	@set -a && . ./configs/local.env && set +a && go run $(MAIN_PKG)

# ─── Testing ──────────────────────────────────────────────────────────────────
test:
	@echo "→ Running unit tests..."
	go test -race -count=1 -timeout=60s ./...

test-verbose:
	go test -v -race -count=1 -timeout=60s ./...

test-integration:
	@echo "→ Running integration tests (requires Docker)..."
	go test -v -race -count=1 -timeout=120s -tags=integration ./tests/integration/...

test-coverage:
	@echo "→ Running tests with coverage..."
	go test -race -count=1 -timeout=60s -coverprofile=$(COVERAGE_OUT) -covermode=atomic ./...
	go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@echo "✓ Coverage report: $(COVERAGE_HTML)"

test-coverage-check:
	@echo "→ Checking coverage threshold ($(COVERAGE_MIN)%)..."
	go test -race -count=1 -timeout=60s -coverprofile=$(COVERAGE_OUT) -covermode=atomic ./...
	@go tool cover -func=$(COVERAGE_OUT) | grep total | awk \
	  '{pct=$$3+0; if (pct < $(COVERAGE_MIN)) {print "FAIL: coverage " $$3 " < $(COVERAGE_MIN)%"; exit 1} else {print "OK: coverage " $$3}}'

# ─── Code Quality ─────────────────────────────────────────────────────────────
fmt:
	@echo "→ Formatting..."
	gofmt -s -w $(GO_FILES)

vet:
	@echo "→ Vetting..."
	go vet ./...

lint:
	@echo "→ Linting..."
	@which golangci-lint > /dev/null || (echo "ERROR: golangci-lint not installed. Run: brew install golangci-lint" && exit 1)
	golangci-lint run ./...

# ─── Code Generation ──────────────────────────────────────────────────────────
proto:
	@echo "→ Generating connect-go from proto files..."
	@which buf > /dev/null || (echo "ERROR: buf not installed. Run: brew install bufbuild/buf/buf" && exit 1)
	buf generate

db-generate:
	@echo "→ Generating sqlc query code..."
	@which sqlc > /dev/null || (echo "ERROR: sqlc not installed. Run: brew install sqlc" && exit 1)
	sqlc generate

mockery:
	@echo "→ Generating interface mocks..."
	@which mockery > /dev/null || (echo "ERROR: mockery not installed. Run: go install github.com/vektra/mockery/v2@latest" && exit 1)
	go generate ./...

# ─── Database ─────────────────────────────────────────────────────────────────
db-migrate:
	@echo "→ Applying schema migrations (Atlas)..."
	@which atlas > /dev/null || (echo "ERROR: atlas not installed. Run: brew install ariga/tap/atlas" && exit 1)
	atlas schema apply --env local --auto-approve

db-rollback:
	@echo "→ Rolling back last migration (Atlas)..."
	atlas migrate down --env local

db-diff:
	@echo "→ Showing schema drift between HCL and database..."
	atlas schema diff --env local

db-status:
	@echo "→ Migration status..."
	atlas migrate status --env local

# ─── OpenBao ──────────────────────────────────────────────────────────────────
openbao-setup:
	@echo "→ Configuring OpenBao dev instance..."
	@which bao > /dev/null || (echo "ERROR: bao CLI not installed. See https://openbao.org/docs/install" && exit 1)
	@export BAO_ADDR=http://127.0.0.1:8200 && export BAO_TOKEN=dev-root-token && \
	  bao secrets enable -path=secret kv-v2 && \
	  bao kv put secret/infinite-brain/dev \
	    jwt_secret="dev-jwt-secret-change-in-prod" \
	    pepper="dev-pepper-change-in-prod" \
	  && echo "✓ OpenBao dev secrets written"

# ─── Docker ───────────────────────────────────────────────────────────────────
docker-up:
	@echo "→ Starting dev services..."
	docker compose up -d
	@echo "✓ Services running. PostgreSQL: 5432, Valkey: 6379, OpenBao: 8200"

docker-down:
	docker compose down

docker-reset:
	docker compose down -v
	docker compose up -d

# ─── Security ─────────────────────────────────────────────────────────────────
security-scan:
	@echo "→ Running security scan..."
	@which gosec > /dev/null || (echo "ERROR: gosec not installed. Run: go install github.com/securego/gosec/v2/cmd/gosec@latest" && exit 1)
	gosec ./...

vuln-check:
	@echo "→ Checking for vulnerabilities..."
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# ─── Cleanup ──────────────────────────────────────────────────────────────────
clean:
	@rm -rf bin/ $(COVERAGE_OUT) $(COVERAGE_HTML)
	@echo "✓ Cleaned"

# ─── Dependencies ─────────────────────────────────────────────────────────────
deps:
	go mod tidy
	go mod verify

deps-upgrade:
	go get -u ./...
	go mod tidy

# ─── Help ─────────────────────────────────────────────────────────────────────
help:
	@echo ""
	@echo "  Infinite Brain — Makefile targets"
	@echo ""
	@echo "  Build:"
	@echo "    make build                  Build the server binary"
	@echo "    make run                    Build and run the server"
	@echo "    make run-local              Build and run with configs/local.env"
	@echo ""
	@echo "  Testing:"
	@echo "    make test                   Run all unit tests"
	@echo "    make test-integration       Run integration tests (needs Docker)"
	@echo "    make test-coverage          Generate coverage HTML report"
	@echo "    make test-coverage-check    Fail if coverage < $(COVERAGE_MIN)%"
	@echo ""
	@echo "  Code Quality:"
	@echo "    make fmt                    Format Go source"
	@echo "    make vet                    Run go vet"
	@echo "    make lint                   Run golangci-lint"
	@echo "    make security-scan          Run gosec security scanner"
	@echo "    make vuln-check             Check for known vulnerabilities"
	@echo ""
	@echo "  Database (Atlas):"
	@echo "    make db-migrate             Apply schema (atlas schema apply)"
	@echo "    make db-rollback            Roll back last migration"
	@echo "    make db-diff                Show drift between HCL schema and database"
	@echo "    make db-status              Show migration status"
	@echo "    make db-generate            Regenerate sqlc query code"
	@echo ""
	@echo "  Code Generation:"
	@echo "    make proto                  Generate connect-go from .proto files (buf)"
	@echo "    make mockery                Regenerate interface mocks"
	@echo ""
	@echo "  Infrastructure:"
	@echo "    make docker-up              Start dev services (PG, Valkey, OpenBao)"
	@echo "    make docker-down            Stop dev services"
	@echo "    make docker-reset           Destroy volumes and restart"
	@echo "    make openbao-setup          Write dev secrets to OpenBao"
	@echo ""
