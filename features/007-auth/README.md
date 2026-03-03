# Feature: Authentication System

**Task ID**: T-007
**Status**: planned
**Epic**: Foundation

## Goal

Implement JWT-based authentication with refresh token rotation.
All API endpoints (except `/health` and `/api/v1/auth/*`) require a valid access token.

## Acceptance Criteria

- [ ] `internal/auth/model.go` — User, Session, Token structs
- [ ] `internal/auth/repository.go` — AuthRepository interface
- [ ] `internal/auth/repository_pg.go` — PostgreSQL implementation
- [ ] `internal/auth/service.go` — AuthService interface
- [ ] `internal/auth/service_impl.go` — Login, register, refresh, logout
- [ ] `internal/auth/handler.go` — HTTP handlers
- [ ] `internal/auth/jwt.go` — JWT generation and validation
- [ ] `pkg/middleware/auth.go` — Auth middleware (extracts user from JWT)
- [ ] `db/migrations/001_create_users.sql`
- [ ] Refresh token rotation (old token invalidated on use)
- [ ] Bcrypt password hashing (cost ≥ 12)
- [ ] Unit tests for all service functions
- [ ] Integration tests for repository
- [ ] HTTP tests for all endpoints

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/auth/register` | Create new user account |
| `POST` | `/api/v1/auth/login` | Login, returns access + refresh tokens |
| `POST` | `/api/v1/auth/refresh` | Exchange refresh token for new token pair |
| `POST` | `/api/v1/auth/logout` | Invalidate refresh token |
| `GET` | `/api/v1/auth/me` | Get current user profile |

## Token Strategy

- **Access token**: 15-minute JWT (HS256, signed with JWT_SECRET)
  - Claims: `user_id`, `email`, `exp`, `iat`, `jti`
- **Refresh token**: opaque UUID stored in PostgreSQL with expiry
  - 7-day validity, single use (rotation on refresh)

## Security Requirements

- Passwords stored as bcrypt (cost 12)
- Refresh tokens are hashed before storage
- JTI (JWT ID) tracked to prevent token reuse
- All auth endpoints rate-limited (10 req/min per IP)
