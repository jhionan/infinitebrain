# Feature: Project Setup

**Task ID**: T-001
**Status**: completed
**Epic**: Foundation

## Goal

Bootstrap the Go project with a production-ready structure, tooling, and conventions so
all future development has a solid base.

## Acceptance Criteria

- [x] `go.mod` initialized with module `github.com/rian/infinite_brain`
- [x] Folder structure created (`cmd/`, `internal/`, `pkg/`, `db/`, `features/`, `tests/`)
- [x] `Makefile` with targets: `build`, `run`, `test`, `lint`, `fmt`, `migrate-*`, `docker-*`
- [x] `docker-compose.yml` with PostgreSQL (pgvector), Redis, MinIO, Jaeger
- [x] `.golangci.yml` with strict linter configuration
- [x] `.github/workflows/ci.yml` with test, lint, build, security jobs
- [x] `configs/example.env` with all required environment variables documented
- [x] `README.md` with quick start instructions
- [x] `docs/IDEA.md`, `docs/ARCHITECTURE.md`, `docs/TASKS.md` created
- [x] zerolog dependency installed

## Implementation Notes

The project uses a modular monolith structure. Each internal package is self-contained with
its own model, repository, service, and handler layers. Packages communicate through
interfaces, enabling easy testing and future extraction into microservices if needed.

Docker Compose uses `pgvector/pgvector:pg16` to support vector embeddings in PostgreSQL
without a separate vector database.
