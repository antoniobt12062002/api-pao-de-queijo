# User Registration & Authentication Design

**Date:** 2026-03-29
**Project:** api-pao-de-queijo
**Status:** Approved

---

## Overview

Add user registration and authentication to the API, supporting both email/password and GitHub OAuth login. The API is for internal use by a development team, so GitHub OAuth is a primary use case. Built with Clean Architecture, GORM, PostgreSQL, golang-migrate, and JWT.

---

## Architecture

Clean Architecture with four layers:

```
handler → usecase → domain ← repository (implements domain interface)
```

- **domain**: defines the `User` entity and `UserRepository` interface — no external dependencies
- **usecase**: orchestrates business logic (create user, authenticate, OAuth)
- **repository/postgres**: GORM implementation of `UserRepository`
- **handler/http**: HTTP handlers, parses requests, calls use cases, returns responses

---

## Project Structure

```
cmd/
  main.go                    ← entry point, dependency wiring
  api.go                     ← HTTP server + router setup

internal/
  domain/
    user.go                  ← User entity + UserRepository interface
  usecase/
    user.go                  ← CreateUser, GetUserByEmail, OAuthLogin
  repository/
    postgres/
      user.go                ← GORM implementation of UserRepository
  handler/
    http/
      user.go                ← POST /v1/users handler
      auth.go                ← login + GitHub OAuth handlers
  db/
    db.go                    ← PostgreSQL connection + GORM setup

migrations/
  000001_create_users_table.up.sql
  000001_create_users_table.down.sql

docker-compose.yml
.env
```

---

## Database Schema

**Table: `users`**

| Column          | Type                      | Constraints                                  |
|-----------------|---------------------------|----------------------------------------------|
| id              | UUID                      | PK, default gen_random_uuid()                |
| name            | VARCHAR(255)              | NOT NULL                                     |
| email           | VARCHAR(255)              | UNIQUE, NOT NULL                             |
| password_hash   | VARCHAR(255)              | NULLABLE (OAuth users have no password)      |
| role            | VARCHAR(50)               | NOT NULL, DEFAULT 'dev'                      |
| phone           | VARCHAR(50)               | NULLABLE                                     |
| provider        | VARCHAR(50)               | NOT NULL, DEFAULT 'local'                    |
| provider_id     | VARCHAR(255)              | NULLABLE (GitHub user ID)                    |
| created_at      | TIMESTAMP WITH TIME ZONE  | NOT NULL                                     |
| updated_at      | TIMESTAMP WITH TIME ZONE  | NOT NULL                                     |

**Indexes:**
- `UNIQUE (email)`
- `UNIQUE (provider, provider_id) WHERE provider_id IS NOT NULL` — prevents duplicate OAuth accounts and enables fast lookup

---

## Endpoints

```
POST /v1/users                    ← register with email/password
POST /v1/auth/login               ← login with email/password, returns JWT
GET  /v1/auth/github              ← redirect to GitHub OAuth
GET  /v1/auth/github/callback     ← GitHub OAuth callback, returns JWT
```

---

## Request / Response Contracts

### POST /v1/users — Register

**Request body:**
```json
{
  "name":     "João Silva",       // required
  "email":    "joao@empresa.com", // required
  "password": "s3nh4segura",      // required, min 8 chars
  "role":     "dev",              // optional, defaults to "dev"
  "phone":    "+5531999999999"    // optional
}
```

**Response 201 Created:**
```json
{
  "id":         "uuid",
  "name":       "João Silva",
  "email":      "joao@empresa.com",
  "role":       "dev",
  "phone":      "+5531999999999",
  "provider":   "local",
  "created_at": "2026-03-29T00:00:00Z"
}
```

### POST /v1/auth/login — Login

**Request body:**
```json
{
  "email":    "joao@empresa.com", // required
  "password": "s3nh4segura"       // required
}
```

**Response 200 OK:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### GET /v1/auth/github — GitHub OAuth redirect
No body. Redirects (302) to GitHub authorization URL with `state` parameter.

### GET /v1/auth/github/callback — GitHub OAuth callback

**Response 200 OK:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### Error envelope (all endpoints)
```json
{
  "error": "descriptive message"
}
```

---

## Authentication Flows

### Email/Password Registration
1. `POST /v1/users` with `name`, `email`, `password`, `role` (optional), `phone` (optional)
2. Validate required fields and password min length (8 chars)
3. Hash password with bcrypt (cost 12)
4. Save user with `provider=local`
5. Return 201 with created user (without `password_hash`)

### Email/Password Login
1. `POST /v1/auth/login` with `email`, `password`
2. Find user by email; if not found → 401
3. Compare bcrypt hash; if mismatch → 401
4. Return signed JWT (see JWT Claims section)

### GitHub OAuth
1. Client calls `GET /v1/auth/github`
2. API generates a random `state` token, stores it in a short-lived server-side store (in-memory map with TTL, keyed by state value)
3. API redirects (302) to `github.com/login/oauth/authorize?client_id=...&state=<state>&scope=user:email`
4. Developer authorizes on GitHub
5. GitHub calls `GET /v1/auth/github/callback?code=xxx&state=xxx`
6. API validates that `state` matches the stored value → if not → 400 Bad Request (CSRF protection)
7. API exchanges `code` for GitHub `access_token` via POST to `github.com/login/oauth/access_token`
8. API fetches developer's GitHub profile (`/user` and `/user/emails`) using the access token
9. **Account collision**: if GitHub email already exists in DB with `provider=local` → return 409 Conflict with message `"email already registered with password login"`
10. If user does not exist → create with `provider=github`, `provider_id=<github_id>`
11. If user exists with `provider=github` → retrieve
12. Return signed JWT

---

## JWT Claims

All tokens are signed with HS256 using `JWT_SECRET`.

```json
{
  "sub":   "user-uuid",
  "email": "joao@empresa.com",
  "role":  "dev",
  "iat":   1711670400,
  "exp":   1711756800
}
```

- `sub`: user UUID
- `exp`: 24 hours from issuance
- Tokens are not refreshable in this iteration (re-login required after expiry)

---

## Infrastructure

### docker-compose.yml
```yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: pao
      POSTGRES_PASSWORD: queijo
      POSTGRES_DB: pao_de_queijo
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

### Environment Variables (.env)
```
DB_DSN=postgres://pao:queijo@localhost:5432/pao_de_queijo?sslmode=disable
GITHUB_CLIENT_ID=xxx
GITHUB_CLIENT_SECRET=xxx
GITHUB_CALLBACK_URL=http://localhost:8080/v1/auth/github/callback
JWT_SECRET=seu-segredo-aqui
```

### Migrations
- Versioned `.sql` files in `migrations/`
- Applied **automatically at app startup** via `golang-migrate`
- If migration fails, the app exits with a fatal error (fail-fast)
- `up` creates the table, `down` drops it

---

## Logging Convention

All logging uses `log/slog` (structured logging). The existing `log.Println` call in `cmd/api.go` will be updated to `slog.Info` as part of this work. New code must use `slog` exclusively.

---

## Go Dependencies

| Package | Purpose |
|---|---|
| `github.com/go-chi/chi/v5` | HTTP router (already in project) |
| `gorm.io/gorm` | ORM |
| `gorm.io/driver/postgres` | PostgreSQL GORM driver |
| `github.com/golang-migrate/migrate/v4` | Versioned migrations |
| `github.com/golang-jwt/jwt/v5` | JWT signing and verification |
| `golang.org/x/crypto` | bcrypt password hashing |
| `github.com/joho/godotenv` | Load .env file |

---

## Error Handling

| Scenario | HTTP Status |
|---|---|
| Duplicate email on registration | 409 Conflict |
| Invalid credentials on login | 401 Unauthorized |
| GitHub OAuth state mismatch (CSRF) | 400 Bad Request |
| GitHub email collides with local account | 409 Conflict |
| GitHub API failure | 502 Bad Gateway |
| Validation errors | 422 Unprocessable Entity |
| User not found | 404 Not Found |

---

## Out of Scope (this iteration)

- Password reset / forgot password
- Email verification
- Refresh tokens
- Role-based authorization middleware
