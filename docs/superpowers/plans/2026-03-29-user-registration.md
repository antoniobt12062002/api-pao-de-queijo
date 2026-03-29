# User Registration & Authentication Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add user registration and GitHub OAuth authentication to the api-pao-de-queijo REST API using Clean Architecture.

**Architecture:** Clean Architecture with four layers — `domain` (entities + interfaces), `usecase` (business logic), `repository/postgres` (GORM implementation), and `handler/http` (HTTP). Dependencies flow inward: handlers call use cases, use cases call repository interfaces defined in domain. PostgreSQL via GORM, versioned migrations with golang-migrate, JWT for auth tokens.

**Tech Stack:** Go, chi, GORM, PostgreSQL, golang-migrate, golang-jwt/jwt/v5, bcrypt, godotenv, Docker Compose.

**Spec:** `docs/superpowers/specs/2026-03-29-user-registration-design.md`

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `docker-compose.yml` | PostgreSQL container |
| Create | `.env` | Environment variables |
| Create | `.env.example` | Committed example env file |
| Create | `migrations/000001_create_users_table.up.sql` | Create users table |
| Create | `migrations/000001_create_users_table.down.sql` | Drop users table |
| Create | `internal/domain/user.go` | User entity + UserRepository interface |
| Create | `internal/db/db.go` | GORM connection + auto-migration runner |
| Create | `internal/repository/postgres/user.go` | GORM implementation of UserRepository |
| Create | `internal/usecase/user.go` | CreateUser, Login, OAuthLogin use cases |
| Create | `internal/handler/http/user.go` | POST /v1/users handler |
| Create | `internal/handler/http/auth.go` | Login + GitHub OAuth handlers |
| Modify | `cmd/api.go` | Register routes, inject dependencies, fix slog |
| Modify | `cmd/main.go` | Load .env, init DB, wire dependencies |

---

## Chunk 1: Infrastructure & Database

---

### Task 1: Docker Compose + environment files

**Files:**
- Create: `docker-compose.yml`
- Create: `.env`
- Create: `.env.example`

- [ ] **Step 1: Create docker-compose.yml**

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

- [ ] **Step 2: Create .env**

```
DB_DSN=postgres://pao:queijo@localhost:5432/pao_de_queijo?sslmode=disable
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
GITHUB_CALLBACK_URL=http://localhost:8080/v1/auth/github/callback
JWT_SECRET=change-me-to-a-long-random-string
```

- [ ] **Step 3: Create .env.example (identical to .env, committed to git)**

Same content as `.env` — this is the file that gets committed. `.env` stays in `.gitignore`.

- [ ] **Step 4: Add .env to .gitignore**

Create `.gitignore` if it doesn't exist, add:
```
.env
```

- [ ] **Step 5: Start the database**

```bash
docker compose up -d
```

Expected: postgres container starts, port 5432 available.

- [ ] **Step 6: Verify connection**

```bash
docker compose ps
```

Expected: `postgres` service status is `running`.

- [ ] **Step 7: Commit**

```bash
git add docker-compose.yml .env.example .gitignore
git commit -m "feat: add docker-compose for postgres and env config"
```

---

### Task 2: Migration files

**Files:**
- Create: `migrations/000001_create_users_table.up.sql`
- Create: `migrations/000001_create_users_table.down.sql`

- [ ] **Step 1: Create up migration**

`migrations/000001_create_users_table.up.sql`:
```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255) NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255),
    role          VARCHAR(50)  NOT NULL DEFAULT 'dev',
    phone         VARCHAR(50),
    provider      VARCHAR(50)  NOT NULL DEFAULT 'local',
    provider_id   VARCHAR(255),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX users_email_idx ON users (email);
CREATE UNIQUE INDEX users_provider_provider_id_idx ON users (provider, provider_id) WHERE provider_id IS NOT NULL;
```

- [ ] **Step 2: Create down migration**

`migrations/000001_create_users_table.down.sql`:
```sql
DROP TABLE IF EXISTS users;
```

- [ ] **Step 3: Commit**

```bash
git add migrations/
git commit -m "feat: add users table migration"
```

---

### Task 3: Install Go dependencies

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add all required packages**

```bash
go get gorm.io/gorm
go get gorm.io/driver/postgres
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto
go get github.com/joho/godotenv
```

- [ ] **Step 2: Tidy**

```bash
go mod tidy
```

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: add gorm, migrate, jwt, bcrypt, godotenv dependencies"
```

---

### Task 4: Database connection + migration runner

**Files:**
- Create: `internal/db/db.go`

- [ ] **Step 1: Create db.go**

```go
package db

import (
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func New(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// NOTE: "file://migrations" is relative to the process working directory.
// Always run the binary from the project root (go run ./cmd/... or ./bin/api from project root).
func RunMigrations(dsn string) error {
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	slog.Info("migrations applied successfully")
	return nil
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/db/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/db/db.go
git commit -m "feat: add database connection and migration runner"
```

---

## Chunk 2: Domain & Repository

---

### Task 5: Domain — User entity and repository interface

**Files:**
- Create: `internal/domain/user.go`

- [ ] **Step 1: Create domain/user.go**

```go
package domain

import "time"

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash *string   `json:"-"`
	Role         string    `json:"role"`
	Phone        *string   `json:"phone"`
	Provider     string    `json:"provider"`
	ProviderID   *string   `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateUserInput struct {
	Name     string
	Email    string
	Password string
	Role     string
	Phone    *string
}

type OAuthUserInput struct {
	Name       string
	Email      string
	Provider   string
	ProviderID string
}

type UserRepository interface {
	Create(user *User) error
	FindByEmail(email string) (*User, error)
	FindByProviderID(provider, providerID string) (*User, error)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/domain/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/domain/user.go
git commit -m "feat: add user domain entity and repository interface"
```

---

### Task 6: Repository — GORM implementation

**Files:**
- Create: `internal/repository/postgres/user.go`

- [ ] **Step 1: Write the failing test**

Create `internal/repository/postgres/user_test.go`:

```go
package postgres_test

import (
	"testing"
	"os"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	userrepo "github.com/antoniobt12062002/pao-de-queijo/internal/repository/postgres"
	"github.com/antoniobt12062002/pao-de-queijo/internal/db"
)

func setupDB(t *testing.T) *userrepo.UserRepository {
	t.Helper()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		t.Skip("DB_DSN not set, skipping integration test")
	}
	gormDB, err := db.New(dsn)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	return userrepo.NewUserRepository(gormDB)
}

func TestCreateAndFindByEmail(t *testing.T) {
	repo := setupDB(t)

	hash := "hashed"
	phone := "+5531999999999"
	user := &domain.User{
		Name:         "Test User",
		Email:        "test_repo@example.com",
		PasswordHash: &hash,
		Role:         "dev",
		Phone:        &phone,
		Provider:     "local",
	}

	err := repo.Create(user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if user.ID == "" {
		t.Fatal("expected ID to be set after create")
	}

	found, err := repo.FindByEmail("test_repo@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if found.Name != "Test User" {
		t.Errorf("expected name 'Test User', got '%s'", found.Name)
	}

	// cleanup
	repo.DB().Exec("DELETE FROM users WHERE email = ?", "test_repo@example.com")
}

func TestFindByProviderID(t *testing.T) {
	repo := setupDB(t)

	pid := "gh_123456"
	user := &domain.User{
		Name:       "GitHub User",
		Email:      "ghuser_repo@example.com",
		Role:       "dev",
		Provider:   "github",
		ProviderID: &pid,
	}

	err := repo.Create(user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := repo.FindByProviderID("github", "gh_123456")
	if err != nil {
		t.Fatalf("FindByProviderID failed: %v", err)
	}
	if found.Email != "ghuser_repo@example.com" {
		t.Errorf("expected email 'ghuser_repo@example.com', got '%s'", found.Email)
	}

	// cleanup
	repo.DB().Exec("DELETE FROM users WHERE email = ?", "ghuser_repo@example.com")
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
DB_DSN="postgres://pao:queijo@localhost:5432/pao_de_queijo?sslmode=disable" go test ./internal/repository/postgres/... -v
```

Expected: FAIL — package does not exist yet.

- [ ] **Step 3: Implement the repository**

Create `internal/repository/postgres/user.go`:

```go
package postgres

import (
	"errors"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"gorm.io/gorm"
)

type userModel struct {
	ID           string    `gorm:"column:id;primaryKey"`
	Name         string    `gorm:"column:name"`
	Email        string    `gorm:"column:email"`
	PasswordHash *string   `gorm:"column:password_hash"`
	Role         string    `gorm:"column:role"`
	Phone        *string   `gorm:"column:phone"`
	Provider     string    `gorm:"column:provider"`
	ProviderID   *string   `gorm:"column:provider_id"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (userModel) TableName() string { return "users" }

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) DB() *gorm.DB { return r.db }

func (r *UserRepository) Create(u *domain.User) error {
	m := toModel(u)
	result := r.db.Create(m)
	if result.Error != nil {
		return result.Error
	}
	u.ID = m.ID
	return nil
}

func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	var m userModel
	result := r.db.Where("email = ?", email).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return toDomain(&m), nil
}

func (r *UserRepository) FindByProviderID(provider, providerID string) (*domain.User, error) {
	var m userModel
	result := r.db.Where("provider = ? AND provider_id = ?", provider, providerID).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return toDomain(&m), nil
}

func toModel(u *domain.User) *userModel {
	return &userModel{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
		Phone:        u.Phone,
		Provider:     u.Provider,
		ProviderID:   u.ProviderID,
	}
}

func toDomain(m *userModel) *domain.User {
	return &domain.User{
		ID:           m.ID,
		Name:         m.Name,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		Role:         m.Role,
		Phone:        m.Phone,
		Provider:     m.Provider,
		ProviderID:   m.ProviderID,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
DB_DSN="postgres://pao:queijo@localhost:5432/pao_de_queijo?sslmode=disable" go test ./internal/repository/postgres/... -v
```

Expected: PASS — both tests green.

- [ ] **Step 5: Commit**

```bash
git add internal/repository/postgres/
git commit -m "feat: add postgres user repository with GORM"
```

---

## Chunk 3: Use Cases

---

### Task 7: Use case — CreateUser

**Files:**
- Create: `internal/usecase/user.go`

- [ ] **Step 1: Write the failing test**

Create `internal/usecase/user_test.go`:

```go
package usecase_test

import (
	"errors"
	"testing"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

// mockRepo implements domain.UserRepository for testing
type mockRepo struct {
	users map[string]*domain.User
}

func newMockRepo() *mockRepo {
	return &mockRepo{users: make(map[string]*domain.User)}
}

func (m *mockRepo) Create(u *domain.User) error {
	u.ID = "test-uuid"
	m.users[u.Email] = u
	return nil
}

func (m *mockRepo) FindByEmail(email string) (*domain.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *mockRepo) FindByProviderID(provider, providerID string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Provider == provider && u.ProviderID != nil && *u.ProviderID == providerID {
			return u, nil
		}
	}
	return nil, nil
}

func TestCreateUser(t *testing.T) {
	repo := newMockRepo()
	uc := usecase.NewUserUseCase(repo, "test-jwt-secret")

	phone := "+5531999999999"
	user, err := uc.CreateUser(domain.CreateUserInput{
		Name:     "João",
		Email:    "joao@example.com",
		Password: "senha123",
		Role:     "dev",
		Phone:    &phone,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if user.ID == "" {
		t.Error("expected user ID to be set")
	}
	if user.PasswordHash != nil {
		t.Error("expected PasswordHash to be nil in returned user")
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	repo := newMockRepo()
	uc := usecase.NewUserUseCase(repo, "test-jwt-secret")

	input := domain.CreateUserInput{
		Name:     "João",
		Email:    "dup@example.com",
		Password: "senha123",
		Role:     "dev",
	}
	_, _ = uc.CreateUser(input)
	_, err := uc.CreateUser(input)
	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
}

func TestOAuthLogin_NewUser(t *testing.T) {
	repo := newMockRepo()
	uc := usecase.NewUserUseCase(repo, "test-jwt-secret")

	token, err := uc.OAuthLogin(domain.OAuthUserInput{
		Name:       "GitHub Dev",
		Email:      "ghdev@example.com",
		Provider:   "github",
		ProviderID: "gh_999",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if token == "" {
		t.Error("expected a JWT token, got empty string")
	}
}

func TestOAuthLogin_EmailTakenByLocalAccount(t *testing.T) {
	repo := newMockRepo()
	uc := usecase.NewUserUseCase(repo, "test-jwt-secret")

	// Create a local account first
	_, err := uc.CreateUser(domain.CreateUserInput{
		Name:     "Local Dev",
		Email:    "local@example.com",
		Password: "senha123",
		Role:     "dev",
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Try to OAuth login with the same email
	_, err = uc.OAuthLogin(domain.OAuthUserInput{
		Name:       "GitHub Dev",
		Email:      "local@example.com",
		Provider:   "github",
		ProviderID: "gh_123",
	})
	if err == nil {
		t.Fatal("expected error for email taken by local account, got nil")
	}
	if !errors.Is(err, usecase.ErrEmailTakenByLocalAccount) {
		t.Errorf("expected ErrEmailTakenByLocalAccount, got: %v", err)
	}
}

```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/usecase/... -v
```

Expected: FAIL — package does not exist yet.

- [ ] **Step 3: Implement CreateUser use case**

Create `internal/usecase/user.go`:

```go
package usecase

import (
	"errors"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists        = errors.New("email already registered")
	ErrEmailTakenByLocalAccount  = errors.New("email already registered with password login")
	ErrInvalidCredentials        = errors.New("invalid email or password")
)

type UserUseCase struct {
	repo      domain.UserRepository
	jwtSecret string
}

func NewUserUseCase(repo domain.UserRepository, jwtSecret string) *UserUseCase {
	return &UserUseCase{repo: repo, jwtSecret: jwtSecret}
}

func (uc *UserUseCase) CreateUser(input domain.CreateUserInput) (*domain.User, error) {
	existing, err := uc.repo.FindByEmail(input.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, err
	}

	role := input.Role
	if role == "" {
		role = "dev"
	}

	hashStr := string(hash)
	user := &domain.User{
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: &hashStr,
		Role:         role,
		Phone:        input.Phone,
		Provider:     "local",
	}

	if err := uc.repo.Create(user); err != nil {
		return nil, err
	}

	// Never return the hash
	user.PasswordHash = nil
	return user, nil
}

func (uc *UserUseCase) Login(email, password string) (string, error) {
	user, err := uc.repo.FindByEmail(email)
	if err != nil {
		return "", err
	}
	if user == nil || user.PasswordHash == nil {
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return uc.signToken(user)
}

func (uc *UserUseCase) OAuthLogin(input domain.OAuthUserInput) (string, error) {
	// Check if email belongs to a local account
	existing, err := uc.repo.FindByEmail(input.Email)
	if err != nil {
		return "", err
	}
	if existing != nil && existing.Provider == "local" {
		return "", ErrEmailTakenByLocalAccount
	}

	// Check if OAuth user already exists
	user, err := uc.repo.FindByProviderID(input.Provider, input.ProviderID)
	if err != nil {
		return "", err
	}

	if user == nil {
		providerID := input.ProviderID
		user = &domain.User{
			Name:       input.Name,
			Email:      input.Email,
			Role:       "dev",
			Provider:   input.Provider,
			ProviderID: &providerID,
		}
		if err := uc.repo.Create(user); err != nil {
			return "", err
		}
	}

	return uc.signToken(user)
}

func (uc *UserUseCase) signToken(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"role":  user.Role,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(uc.jwtSecret))
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/usecase/... -v
```

Expected: PASS — all tests green.

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/
git commit -m "feat: add user use cases (create, login, oauth login)"
```

---

## Chunk 4: HTTP Handlers & Wiring

---

### Task 8: User registration handler

**Files:**
- Create: `internal/handler/http/user.go`

- [ ] **Step 1: Write the failing test**

Create `internal/handler/http/user_test.go`:

```go
package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	handler "github.com/antoniobt12062002/pao-de-queijo/internal/handler/http"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

type stubRepo struct{ users map[string]*domain.User }

func newStubRepo() *stubRepo { return &stubRepo{users: make(map[string]*domain.User)} }
func (s *stubRepo) Create(u *domain.User) error          { u.ID = "uuid-1"; s.users[u.Email] = u; return nil }
func (s *stubRepo) FindByEmail(e string) (*domain.User, error) { u := s.users[e]; return u, nil }
func (s *stubRepo) FindByProviderID(p, id string) (*domain.User, error) { return nil, nil }

func TestRegisterHandler(t *testing.T) {
	uc := usecase.NewUserUseCase(newStubRepo(), "secret")
	h := handler.NewUserHandler(uc)

	body := `{"name":"João","email":"joao@test.com","password":"senha123","role":"dev"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["email"] != "joao@test.com" {
		t.Errorf("expected email in response, got: %v", resp)
	}
	if _, ok := resp["password_hash"]; ok {
		t.Error("password_hash must not be in response")
	}
}

func TestRegisterHandler_DuplicateEmail(t *testing.T) {
	repo := newStubRepo()
	uc := usecase.NewUserUseCase(repo, "secret")
	h := handler.NewUserHandler(uc)

	body := `{"name":"João","email":"dup@test.com","password":"senha123","role":"dev"}`
	req1 := httptest.NewRequest(http.MethodPost, "/v1/users", bytes.NewBufferString(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	h.Register(w1, req1)

	req2 := httptest.NewRequest(http.MethodPost, "/v1/users", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	h.Register(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w2.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/handler/http/... -v
```

Expected: FAIL — package does not exist yet.

- [ ] **Step 3: Implement user handler**

Create `internal/handler/http/user.go`:

```go
package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

type UserHandler struct {
	uc *usecase.UserUseCase
}

func NewUserHandler(uc *usecase.UserUseCase) *UserHandler {
	return &UserHandler{uc: uc}
}

type registerRequest struct {
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Role     string  `json:"role"`
	Phone    *string `json:"phone"`
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusUnprocessableEntity, "name, email and password are required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusUnprocessableEntity, "password must be at least 8 characters")
		return
	}

	user, err := h.uc.CreateUser(domain.CreateUserInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
		Phone:    req.Phone,
	})
	if err != nil {
		if errors.Is(err, usecase.ErrEmailAlreadyExists) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/handler/http/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/handler/http/user.go internal/handler/http/user_test.go
git commit -m "feat: add user registration HTTP handler"
```

---

### Task 9: Auth handlers (login + GitHub OAuth)

**Files:**
- Create: `internal/handler/http/auth.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/handler/http/auth_test.go`:

```go
package http_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/antoniobt12062002/pao-de-queijo/internal/handler/http"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	uc := usecase.NewUserUseCase(newStubRepo(), "secret")
	h := handler.NewAuthHandler(uc, "", "", "")

	body := `{"email":"nobody@test.com","password":"wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGitHubRedirect(t *testing.T) {
	uc := usecase.NewUserUseCase(newStubRepo(), "secret")
	h := handler.NewAuthHandler(uc, "client-id", "client-secret", "http://localhost/callback")

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/github", nil)
	w := httptest.NewRecorder()

	h.GitHubLogin(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc == "" {
		t.Error("expected Location header in redirect")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/handler/http/... -v
```

Expected: FAIL — auth.go doesn't exist yet.

- [ ] **Step 3: Implement auth handler**

Create `internal/handler/http/auth.go`:

```go
package http

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

type stateEntry struct {
	createdAt time.Time
}

type AuthHandler struct {
	uc               *usecase.UserUseCase
	githubClientID   string
	githubSecret     string
	githubCallbackURL string
	states           map[string]stateEntry
	mu               sync.Mutex
}

func NewAuthHandler(uc *usecase.UserUseCase, clientID, secret, callbackURL string) *AuthHandler {
	return &AuthHandler{
		uc:               uc,
		githubClientID:   clientID,
		githubSecret:     secret,
		githubCallbackURL: callbackURL,
		states:           make(map[string]stateEntry),
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	token, err := h.uc.Login(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *AuthHandler) GitHubLogin(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	h.mu.Lock()
	h.states[state] = stateEntry{createdAt: time.Now()}
	h.mu.Unlock()

	redirectURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=user:email&state=%s",
		h.githubClientID,
		url.QueryEscape(h.githubCallbackURL),
		state,
	)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *AuthHandler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	h.mu.Lock()
	entry, ok := h.states[state]
	if ok {
		delete(h.states, state)
	}
	h.mu.Unlock()

	if !ok || time.Since(entry.createdAt) > 10*time.Minute {
		writeError(w, http.StatusBadRequest, "invalid or expired state parameter")
		return
	}

	accessToken, err := h.exchangeGitHubCode(code)
	if err != nil {
		slog.Error("github token exchange failed", "err", err)
		writeError(w, http.StatusBadGateway, "failed to authenticate with GitHub")
		return
	}

	ghUser, err := h.fetchGitHubUser(accessToken)
	if err != nil {
		slog.Error("github user fetch failed", "err", err)
		writeError(w, http.StatusBadGateway, "failed to fetch GitHub user")
		return
	}

	token, err := h.uc.OAuthLogin(domain.OAuthUserInput{
		Name:       ghUser.Name,
		Email:      ghUser.Email,
		Provider:   "github",
		ProviderID: fmt.Sprintf("%d", ghUser.ID),
	})
	if err != nil {
		if errors.Is(err, usecase.ErrEmailTakenByLocalAccount) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

type githubUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (h *AuthHandler) exchangeGitHubCode(code string) (string, error) {
	body := fmt.Sprintf("client_id=%s&client_secret=%s&code=%s",
		h.githubClientID, h.githubSecret, code)

	req, _ := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token",
		strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("github error: %s", result.Error)
	}
	return result.AccessToken, nil
}

func (h *AuthHandler) fetchGitHubUser(accessToken string) (*githubUser, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var u githubUser
	if err := json.Unmarshal(data, &u); err != nil {
		return nil, err
	}

	// If email is not public, fetch from /user/emails
	if u.Email == "" {
		u.Email, err = h.fetchPrimaryEmail(accessToken)
		if err != nil {
			return nil, err
		}
	}
	return &u, nil
}

func (h *AuthHandler) fetchPrimaryEmail(accessToken string) (string, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}
	for _, e := range emails {
		if e.Primary {
			return e.Email, nil
		}
	}
	return "", fmt.Errorf("no primary email found on GitHub account")
}

func generateState() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(b)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/handler/http/... -v
```

Expected: PASS — all handler tests green.

- [ ] **Step 5: Commit**

```bash
git add internal/handler/http/auth.go internal/handler/http/auth_test.go
git commit -m "feat: add login and GitHub OAuth handlers"
```

---

### Task 10: Wire everything in cmd/main.go and cmd/api.go

**Files:**
- Modify: `cmd/api.go`
- Modify: `cmd/main.go`

> **Important:** Update `cmd/api.go` FIRST (Step 1). The new `main.go` references `application.userHandler` and `application.authHandler` which are defined in `api.go`. If you write `main.go` first the build will fail. Always update `api.go` → `main.go` → build.

- [ ] **Step 1: Update cmd/api.go**

Replace the content of `cmd/api.go` with:

```go
package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	handler "github.com/antoniobt12062002/pao-de-queijo/internal/handler/http"
)

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("all good"))
	})

	r.Route("/v1", func(r chi.Router) {
		r.Post("/users", app.userHandler.Register)
		r.Post("/auth/login", app.authHandler.Login)
		r.Get("/auth/github", app.authHandler.GitHubLogin)
		r.Get("/auth/github/callback", app.authHandler.GitHubCallback)
	})

	return r
}

func (app *application) run(h http.Handler) error {
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      h,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	slog.Info("starting server", "addr", app.config.addr)
	return srv.ListenAndServe()
}

type application struct {
	config      config
	userHandler *handler.UserHandler
	authHandler *handler.AuthHandler
}

type config struct {
	addr string
	db   dbConfig
}

type dbConfig struct {
	dsn string
}
```

- [ ] **Step 2: Update cmd/main.go**

Replace the content of `cmd/main.go` with:

```go
package main

import (
	"log/slog"
	"os"

	"github.com/antoniobt12062002/pao-de-queijo/internal/db"
	handler "github.com/antoniobt12062002/pao-de-queijo/internal/handler/http"
	"github.com/antoniobt12062002/pao-de-queijo/internal/repository/postgres"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
	"github.com/joho/godotenv"
)

func main() {
	// Setup logger first so all subsequent slog calls are structured
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found, using environment variables")
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		slog.Error("DB_DSN environment variable is required")
		os.Exit(1)
	}

	if err := db.RunMigrations(dsn); err != nil {
		slog.Error("failed to run migrations", "err", err)
		os.Exit(1)
	}

	gormDB, err := db.New(dsn)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}

	userRepo := postgres.NewUserRepository(gormDB)
	userUC := usecase.NewUserUseCase(userRepo, os.Getenv("JWT_SECRET"))
	userHandler := handler.NewUserHandler(userUC)
	authHandler := handler.NewAuthHandler(
		userUC,
		os.Getenv("GITHUB_CLIENT_ID"),
		os.Getenv("GITHUB_CLIENT_SECRET"),
		os.Getenv("GITHUB_CALLBACK_URL"),
	)

	cfg := &config{
		addr: ":8080",
		db:   dbConfig{dsn: dsn},
	}

	api := &application{
		config:      *cfg,
		userHandler: userHandler,
		authHandler: authHandler,
	}

	if err := api.run(api.mount()); err != nil {
		slog.Error("error starting server", "err", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Build the full project**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Run all tests**

```bash
go test ./... -v
```

Expected: all tests pass (integration tests skipped if DB_DSN not set).

- [ ] **Step 5: Start the server and smoke test**

```bash
go run ./cmd/...
```

In another terminal:
```bash
curl -s http://localhost:8080/health
```
Expected: `all good`

```bash
curl -s -X POST http://localhost:8080/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Dev Test","email":"dev@test.com","password":"senha123","role":"dev"}'
```
Expected: 201 with user JSON (no `password_hash`).

- [ ] **Step 6: Commit**

```bash
git add cmd/main.go cmd/api.go
git commit -m "feat: wire dependencies and register all routes"
```

---

## Chunk 5: Swagger / OpenAPI

---

### Task 11: Configurar swaggo/swag

**Files:**
- Modify: `go.mod`, `go.sum`
- Modify: `cmd/api.go` ← adicionar rota `/swagger/*`
- Modify: `cmd/main.go` ← adicionar import dos docs gerados
- Modify: `internal/handler/http/user.go` ← anotações swag
- Modify: `internal/handler/http/auth.go` ← anotações swag
- Create: `docs/` ← gerado automaticamente pelo swag CLI

- [ ] **Step 1: Instalar swag CLI**

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Verificar instalação:
```bash
swag --version
```
Expected: `swag version vX.X.X`

- [ ] **Step 2: Adicionar dependências Go**

```bash
go get github.com/swaggo/swag
go get github.com/swaggo/http-swagger
go get github.com/swaggo/files
go mod tidy
```

- [ ] **Step 3: Adicionar anotação geral da API em cmd/main.go**

Adicione o bloco de comentário antes da função `main`:

```go
// @title           API Pão de Queijo
// @version         1.0
// @description     API interna para o time de desenvolvimento.
// @host            localhost:8080
// @BasePath        /v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
```

- [ ] **Step 4: Adicionar anotações ao handler de registro (internal/handler/http/user.go)**

Adicione o bloco de comentário acima de `func (h *UserHandler) Register`:

```go
// Register godoc
// @Summary      Cadastrar usuário
// @Description  Cria um novo usuário com email e senha
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      registerRequest       true  "Dados do usuário"
// @Success      201   {object}  domain.User
// @Failure      409   {object}  map[string]string     "Email já cadastrado"
// @Failure      422   {object}  map[string]string     "Dados inválidos"
// @Router       /users [post]
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
```

- [ ] **Step 5: Adicionar anotações ao handler de login (internal/handler/http/auth.go)**

Acima de `func (h *AuthHandler) Login`:

```go
// Login godoc
// @Summary      Login com email e senha
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest          true  "Credenciais"
// @Success      200   {object}  map[string]string     "token JWT"
// @Failure      401   {object}  map[string]string     "Credenciais inválidas"
// @Failure      422   {object}  map[string]string     "Dados inválidos"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
```

Acima de `func (h *AuthHandler) GitHubLogin`:

```go
// GitHubLogin godoc
// @Summary      Iniciar login com GitHub
// @Description  Redireciona para o GitHub OAuth
// @Tags         auth
// @Success      302  "Redireciona para GitHub"
// @Router       /auth/github [get]
func (h *AuthHandler) GitHubLogin(w http.ResponseWriter, r *http.Request) {
```

Acima de `func (h *AuthHandler) GitHubCallback`:

```go
// GitHubCallback godoc
// @Summary      Callback do GitHub OAuth
// @Description  GitHub chama este endpoint após autorização
// @Tags         auth
// @Produce      json
// @Param        code   query  string  true  "Código do GitHub"
// @Param        state  query  string  true  "State anti-CSRF"
// @Success      200    {object}  map[string]string  "token JWT"
// @Failure      400    {object}  map[string]string  "State inválido"
// @Failure      502    {object}  map[string]string  "Erro no GitHub"
// @Router       /auth/github/callback [get]
func (h *AuthHandler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
```

Adicione também a struct `loginRequest` em `auth.go` para o swag gerar o schema correto:

```go
type loginRequest struct {
	Email    string `json:"email"    example:"joao@empresa.com"`
	Password string `json:"password" example:"s3nh4segura"`
}
```

- [ ] **Step 6: Gerar os docs**

Execute a partir da raiz do projeto:

```bash
swag init -g cmd/main.go --output docs/swagger
```

Expected: pasta `docs/swagger/` criada com `docs.go`, `swagger.json`, `swagger.yaml`.

- [ ] **Step 7: Registrar a rota do Swagger em cmd/api.go**

Adicione o import e a rota no método `mount()`:

```go
import (
    // ... imports existentes ...
    httpSwagger "github.com/swaggo/http-swagger"
    _ "github.com/antoniobt12062002/pao-de-queijo/docs/swagger" // docs gerados
)

// dentro de mount(), antes do return r:
r.Get("/swagger/*", httpSwagger.Handler(
    httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
))
```

- [ ] **Step 8: Build e verificar**

```bash
go build ./...
```

Expected: sem erros.

- [ ] **Step 9: Testar a UI do Swagger**

```bash
go run ./cmd/...
```

Abra no browser: `http://localhost:8080/swagger/index.html`

Expected: UI do Swagger com os 4 endpoints documentados.

- [ ] **Step 10: Commit**

```bash
git add docs/swagger/ cmd/main.go cmd/api.go internal/handler/http/user.go internal/handler/http/auth.go go.mod go.sum
git commit -m "feat: add swagger/openapi documentation with swaggo"
```
