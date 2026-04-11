# Feature: Participation — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implementar o módulo de participação em rodadas (`POST /v1/rounds/:id/participate`, `DELETE /v1/rounds/:id/participate`, `GET /v1/rounds/:id/participations`) com regras de status e notificação de rodada aberta ao confirmar pagamento.

**Architecture:** Segue o padrão existente (domain → usecase → postgres repository → chi handler). A única dependência de feature anterior é `feature/round` (já mergeada). O `RoundUseCase.Confirm` recebe uma adição: ao abrir a rodada, dispara `NotificationService.SendParticipationOpen(userIDs)` com os membros da rotação. Nenhum novo job é necessário.

**Tech Stack:** Go, GORM, golang-migrate, chi, JWT middleware, go test (stdlib)

**GitHub Issue:** #5 — Branch: `feature/participation`

---

## Arquivo de Referência do Spec

`docs/superpowers/specs/2026-03-29-pao-de-queijo-design.md` — seção: Features > Feature 4

---

## Mapa de Arquivos

| Arquivo | Ação | Responsabilidade |
|---------|------|-----------------|
| `migrations/000005_create_participations_table.up.sql` | Criar | Tabela `participations` |
| `migrations/000005_create_participations_table.down.sql` | Criar | Rollback |
| `internal/domain/participation.go` | Criar | `Participation`, `ParticipationRepository` interface, erros |
| `internal/domain/notification.go` | Modificar | Adicionar `SendParticipationOpen(userIDs []string) error` à interface e noop |
| `internal/usecase/round_test.go` | Modificar | Adicionar `SendParticipationOpen` ao `mockNotifySvc` |
| `internal/handler/http/round_test.go` | Modificar | Adicionar `SendParticipationOpen` ao `stubNotifySvcForRound` |
| `internal/repository/postgres/participation.go` | Criar | `ParticipationRepository` postgres com GORM |
| `internal/usecase/participation.go` | Criar | `ParticipationUseCase`: Participate, Withdraw, GetParticipations |
| `internal/usecase/participation_test.go` | Criar | Testes unitários |
| `internal/usecase/round.go` | Modificar | `Confirm` dispara `SendParticipationOpen` após abrir rodada |
| `internal/handler/http/participation.go` | Criar | `ParticipationHandler`: 3 endpoints com Swagger |
| `cmd/main.go` | Modificar | Wiring: participationRepo → participationUC → participationHandler |
| `cmd/api.go` | Modificar | Adicionar `participationHandler` ao struct + registrar 3 rotas |

---

## Chunk 1: Migração, Domain e Repository

### Task 1: Criar migração

**Files:**
- Create: `migrations/000005_create_participations_table.up.sql`
- Create: `migrations/000005_create_participations_table.down.sql`

- [ ] **Step 1: Criar branch**

```bash
git checkout -b feature/participation
```

- [ ] **Step 2: Criar migração up**

Arquivo `migrations/000005_create_participations_table.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS participations (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    round_id     UUID        NOT NULL REFERENCES rounds(id),
    user_id      UUID        NOT NULL REFERENCES users(id),
    quantity     INT         NOT NULL CHECK (quantity >= 1),
    confirmed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (round_id, user_id)
);
```

- [ ] **Step 3: Criar migração down**

Arquivo `migrations/000005_create_participations_table.down.sql`:

```sql
DROP TABLE IF EXISTS participations;
```

- [ ] **Step 4: Commit**

```bash
git add migrations/
git commit -m "feat(participation): add participations table migration"
```

---

### Task 2: Domain participation.go

**Files:**
- Create: `internal/domain/participation.go`

- [ ] **Step 1: Criar domain/participation.go**

```go
package domain

import (
	"errors"
	"time"
)

var (
	ErrParticipationNotFound  = errors.New("participation not found")
	ErrRoundNotOpen           = errors.New("round is not open")
	ErrAlreadyParticipating   = errors.New("already participating in this round")
)

type Participation struct {
	ID          string    `json:"id"`
	RoundID     string    `json:"round_id"`
	UserID      string    `json:"user_id"`
	Quantity    int       `json:"quantity"`
	ConfirmedAt time.Time `json:"confirmed_at"`
}

type ParticipationRepository interface {
	Create(p *Participation) error
	GetByRoundAndUser(roundID, userID string) (*Participation, error)
	GetByRound(roundID string) ([]*Participation, error)
	Delete(roundID, userID string) error
}
```

- [ ] **Step 2: Verificar compilação**

```bash
go build ./...
```

Esperado: sem erros

- [ ] **Step 3: Commit**

```bash
git add internal/domain/participation.go
git commit -m "feat(participation): add Participation domain and repository interface"
```

---

### Task 3: Atualizar NotificationService e mocks existentes

**Files:**
- Modify: `internal/domain/notification.go`
- Modify: `internal/usecase/round_test.go`
- Modify: `internal/handler/http/round_test.go`

- [ ] **Step 1: Adicionar SendParticipationOpen à interface e noop**

Em `internal/domain/notification.go`, adicionar o método à interface e ao noop:

```go
package domain

// NotificationService envia notificações aos usuários.
// Implementação real será feita em feature/notification.
type NotificationService interface {
	SendRoundAnnounced(payerID string) error
	SendRoundClosed(payerID string) error
	SendReminder(participantIDs []string) error
	SendParticipationOpen(userIDs []string) error
}

// NoopNotificationService é a implementação stub usada até feature/notification.
type NoopNotificationService struct{}

func (n *NoopNotificationService) SendRoundAnnounced(payerID string) error       { return nil }
func (n *NoopNotificationService) SendRoundClosed(payerID string) error          { return nil }
func (n *NoopNotificationService) SendReminder(participantIDs []string) error    { return nil }
func (n *NoopNotificationService) SendParticipationOpen(userIDs []string) error  { return nil }
```

- [ ] **Step 2: Verificar que a compilação falha (mocks desatualizados)**

```bash
go build ./...
```

Esperado: erros de compilação nos arquivos de teste sobre `mockNotifySvc` e `stubNotifySvcForRound` não implementando a interface.

- [ ] **Step 3: Atualizar mockNotifySvc em internal/usecase/round_test.go**

Localizar o bloco `type mockNotifySvc struct{}` e adicionar o método:

```go
func (n *mockNotifySvc) SendParticipationOpen(ids []string) error { return nil }
```

- [ ] **Step 4: Atualizar stubNotifySvcForRound em internal/handler/http/round_test.go**

Localizar o bloco `type stubNotifySvcForRound struct{}` e adicionar o método:

```go
func (s *stubNotifySvcForRound) SendParticipationOpen(ids []string) error { return nil }
```

- [ ] **Step 5: Verificar compilação e testes existentes passam**

```bash
go build ./...
go test ./internal/usecase/... ./internal/handler/http/...
```

Esperado: compilação limpa, todos os testes existentes passam

- [ ] **Step 6: Commit**

```bash
git add internal/domain/notification.go internal/usecase/round_test.go internal/handler/http/round_test.go
git commit -m "feat(participation): add SendParticipationOpen to NotificationService interface"
```

---

### Task 4: Postgres repository

**Files:**
- Create: `internal/repository/postgres/participation.go`

- [ ] **Step 1: Criar postgres/participation.go**

```go
package postgres

import (
	"errors"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"gorm.io/gorm"
)

type participationModel struct {
	ID          string    `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	RoundID     string    `gorm:"column:round_id;type:uuid"`
	UserID      string    `gorm:"column:user_id;type:uuid"`
	Quantity    int       `gorm:"column:quantity"`
	ConfirmedAt time.Time `gorm:"column:confirmed_at;autoCreateTime"`
}

func (participationModel) TableName() string { return "participations" }

type ParticipationRepository struct {
	db *gorm.DB
}

func NewParticipationRepository(db *gorm.DB) *ParticipationRepository {
	return &ParticipationRepository{db: db}
}

func (r *ParticipationRepository) Create(p *domain.Participation) error {
	m := &participationModel{
		RoundID:  p.RoundID,
		UserID:   p.UserID,
		Quantity: p.Quantity,
	}
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	p.ID = m.ID
	return nil
}

func (r *ParticipationRepository) GetByRoundAndUser(roundID, userID string) (*domain.Participation, error) {
	var m participationModel
	result := r.db.Where("round_id = ? AND user_id = ?", roundID, userID).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return toDomainParticipation(&m), nil
}

func (r *ParticipationRepository) GetByRound(roundID string) ([]*domain.Participation, error) {
	var models []participationModel
	if err := r.db.Where("round_id = ?", roundID).Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.Participation, len(models))
	for i, m := range models {
		result[i] = toDomainParticipation(&m)
	}
	return result, nil
}

func (r *ParticipationRepository) Delete(roundID, userID string) error {
	return r.db.Where("round_id = ? AND user_id = ?", roundID, userID).Delete(&participationModel{}).Error
}

func toDomainParticipation(m *participationModel) *domain.Participation {
	return &domain.Participation{
		ID:          m.ID,
		RoundID:     m.RoundID,
		UserID:      m.UserID,
		Quantity:    m.Quantity,
		ConfirmedAt: m.ConfirmedAt,
	}
}

// Verify interface compliance at compile time
var _ domain.ParticipationRepository = (*ParticipationRepository)(nil)
```

- [ ] **Step 2: Verificar compilação**

```bash
go build ./...
```

Esperado: sem erros

- [ ] **Step 3: Commit**

```bash
git add internal/repository/postgres/participation.go
git commit -m "feat(participation): add ParticipationRepository postgres implementation"
```

---

## Chunk 2: UseCase com TDD

### Task 5: ParticipationUseCase com testes

**Files:**
- Create: `internal/usecase/participation_test.go`
- Create: `internal/usecase/participation.go`

- [ ] **Step 1: Escrever os testes (failing)**

Arquivo `internal/usecase/participation_test.go`:

```go
package usecase_test

import (
	"testing"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

// --- mocks ---

type mockParticipationRepo struct {
	byRoundUser map[string]*domain.Participation // key: roundID+":"+userID
	byRound     map[string][]*domain.Participation
}

func newMockParticipationRepo() *mockParticipationRepo {
	return &mockParticipationRepo{
		byRoundUser: make(map[string]*domain.Participation),
		byRound:     make(map[string][]*domain.Participation),
	}
}

func (m *mockParticipationRepo) Create(p *domain.Participation) error {
	key := p.RoundID + ":" + p.UserID
	if _, exists := m.byRoundUser[key]; exists {
		return domain.ErrAlreadyParticipating
	}
	p.ID = "part-uuid-1"
	p.ConfirmedAt = time.Now()
	m.byRoundUser[key] = p
	m.byRound[p.RoundID] = append(m.byRound[p.RoundID], p)
	return nil
}

func (m *mockParticipationRepo) GetByRoundAndUser(roundID, userID string) (*domain.Participation, error) {
	key := roundID + ":" + userID
	return m.byRoundUser[key], nil
}

func (m *mockParticipationRepo) GetByRound(roundID string) ([]*domain.Participation, error) {
	return m.byRound[roundID], nil
}

func (m *mockParticipationRepo) Delete(roundID, userID string) error {
	key := roundID + ":" + userID
	if _, exists := m.byRoundUser[key]; !exists {
		return domain.ErrParticipationNotFound
	}
	p := m.byRoundUser[key]
	delete(m.byRoundUser, key)
	list := m.byRound[roundID]
	for i, item := range list {
		if item.UserID == userID {
			m.byRound[roundID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	_ = p
	return nil
}

func openRound(id string) *domain.Round {
	return &domain.Round{ID: id, Date: "2026-01-01", PayerID: "payer-1", Status: domain.RoundStatusOpen}
}

func pendingRound(id string) *domain.Round {
	return &domain.Round{ID: id, Date: "2026-01-01", PayerID: "payer-1", Status: domain.RoundStatusPending}
}

// --- tests ---

func TestParticipationUseCase_Participate_OK(t *testing.T) {
	roundRepo := newMockRoundRepo()
	round := openRound("round-1")
	roundRepo.rounds["round-1"] = round

	uc := usecase.NewParticipationUseCase(newMockParticipationRepo(), roundRepo)
	if err := uc.Participate("round-1", "user-1", 2); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestParticipationUseCase_Participate_RoundNotFound(t *testing.T) {
	uc := usecase.NewParticipationUseCase(newMockParticipationRepo(), newMockRoundRepo())
	err := uc.Participate("nonexistent", "user-1", 1)
	if err != domain.ErrRoundNotFound {
		t.Errorf("expected ErrRoundNotFound, got: %v", err)
	}
}

func TestParticipationUseCase_Participate_RoundNotOpen(t *testing.T) {
	roundRepo := newMockRoundRepo()
	roundRepo.rounds["round-1"] = pendingRound("round-1")

	uc := usecase.NewParticipationUseCase(newMockParticipationRepo(), roundRepo)
	err := uc.Participate("round-1", "user-1", 1)
	if err != domain.ErrRoundNotOpen {
		t.Errorf("expected ErrRoundNotOpen, got: %v", err)
	}
}

func TestParticipationUseCase_Participate_AlreadyParticipating(t *testing.T) {
	roundRepo := newMockRoundRepo()
	roundRepo.rounds["round-1"] = openRound("round-1")
	partRepo := newMockParticipationRepo()

	uc := usecase.NewParticipationUseCase(partRepo, roundRepo)
	_ = uc.Participate("round-1", "user-1", 1)
	err := uc.Participate("round-1", "user-1", 1)
	if err != domain.ErrAlreadyParticipating {
		t.Errorf("expected ErrAlreadyParticipating, got: %v", err)
	}
}

func TestParticipationUseCase_Withdraw_OK(t *testing.T) {
	roundRepo := newMockRoundRepo()
	roundRepo.rounds["round-1"] = openRound("round-1")
	partRepo := newMockParticipationRepo()

	uc := usecase.NewParticipationUseCase(partRepo, roundRepo)
	_ = uc.Participate("round-1", "user-1", 1)
	if err := uc.Withdraw("round-1", "user-1"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestParticipationUseCase_Withdraw_RoundNotOpen(t *testing.T) {
	roundRepo := newMockRoundRepo()
	roundRepo.rounds["round-1"] = pendingRound("round-1")

	uc := usecase.NewParticipationUseCase(newMockParticipationRepo(), roundRepo)
	err := uc.Withdraw("round-1", "user-1")
	if err != domain.ErrRoundNotOpen {
		t.Errorf("expected ErrRoundNotOpen, got: %v", err)
	}
}

func TestParticipationUseCase_Withdraw_NotFound(t *testing.T) {
	roundRepo := newMockRoundRepo()
	roundRepo.rounds["round-1"] = openRound("round-1")

	uc := usecase.NewParticipationUseCase(newMockParticipationRepo(), roundRepo)
	err := uc.Withdraw("round-1", "user-1")
	if err != domain.ErrParticipationNotFound {
		t.Errorf("expected ErrParticipationNotFound, got: %v", err)
	}
}

func TestParticipationUseCase_GetParticipations_Empty(t *testing.T) {
	uc := usecase.NewParticipationUseCase(newMockParticipationRepo(), newMockRoundRepo())
	resp, err := uc.GetParticipations("round-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("expected empty data, got %d", len(resp.Data))
	}
	if resp.TotalQuantity != 0 {
		t.Errorf("expected total_quantity 0, got %d", resp.TotalQuantity)
	}
}

func TestParticipationUseCase_GetParticipations_WithData(t *testing.T) {
	roundRepo := newMockRoundRepo()
	roundRepo.rounds["round-1"] = openRound("round-1")
	partRepo := newMockParticipationRepo()

	uc := usecase.NewParticipationUseCase(partRepo, roundRepo)
	_ = uc.Participate("round-1", "user-1", 3)
	_ = uc.Participate("round-1", "user-2", 2)

	resp, err := uc.GetParticipations("round-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 participations, got %d", len(resp.Data))
	}
	if resp.TotalQuantity != 5 {
		t.Errorf("expected total_quantity 5, got %d", resp.TotalQuantity)
	}
}
```

- [ ] **Step 2: Rodar para verificar que falha**

```bash
go test ./internal/usecase/... -run TestParticipation -v
```

Esperado: FAIL com "undefined: usecase.NewParticipationUseCase"

- [ ] **Step 3: Implementar ParticipationUseCase**

Arquivo `internal/usecase/participation.go`:

```go
package usecase

import "github.com/antoniobt12062002/pao-de-queijo/internal/domain"

type ParticipationUseCase struct {
	partRepo  domain.ParticipationRepository
	roundRepo domain.RoundRepository
}

func NewParticipationUseCase(
	partRepo domain.ParticipationRepository,
	roundRepo domain.RoundRepository,
) *ParticipationUseCase {
	return &ParticipationUseCase{partRepo: partRepo, roundRepo: roundRepo}
}

type ParticipationsResponse struct {
	Data          []*domain.Participation `json:"data"`
	TotalQuantity int                     `json:"total_quantity"`
}

func (uc *ParticipationUseCase) Participate(roundID, userID string, quantity int) error {
	round, err := uc.roundRepo.GetByID(roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return domain.ErrRoundNotFound
	}
	if round.Status != domain.RoundStatusOpen {
		return domain.ErrRoundNotOpen
	}
	return uc.partRepo.Create(&domain.Participation{
		RoundID:  roundID,
		UserID:   userID,
		Quantity: quantity,
	})
}

func (uc *ParticipationUseCase) Withdraw(roundID, userID string) error {
	round, err := uc.roundRepo.GetByID(roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return domain.ErrRoundNotFound
	}
	if round.Status != domain.RoundStatusOpen {
		return domain.ErrRoundNotOpen
	}
	existing, err := uc.partRepo.GetByRoundAndUser(roundID, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrParticipationNotFound
	}
	return uc.partRepo.Delete(roundID, userID)
}

func (uc *ParticipationUseCase) GetParticipations(roundID string) (*ParticipationsResponse, error) {
	parts, err := uc.partRepo.GetByRound(roundID)
	if err != nil {
		return nil, err
	}
	total := 0
	for _, p := range parts {
		total += p.Quantity
	}
	return &ParticipationsResponse{
		Data:          parts,
		TotalQuantity: total,
	}, nil
}
```

- [ ] **Step 4: Rodar para verificar que passa**

```bash
go test ./internal/usecase/... -run TestParticipation -v
```

Esperado: todos os 9 testes PASS

- [ ] **Step 5: Rodar todos os testes para garantir nenhuma regressão**

```bash
go test ./internal/usecase/...
```

Esperado: ok (incluindo testes do RoundUseCase)

- [ ] **Step 6: Commit**

```bash
git add internal/usecase/participation.go internal/usecase/participation_test.go
git commit -m "feat(participation): add ParticipationUseCase with TDD"
```

---

## Chunk 3: Handler, Atualização do RoundUseCase e Wiring

### Task 6: Atualizar RoundUseCase.Confirm para notificar participação aberta

**Files:**
- Modify: `internal/usecase/round.go`

- [ ] **Step 1: Atualizar Confirm em internal/usecase/round.go**

Substituir o método `Confirm` existente:

```go
func (uc *RoundUseCase) Confirm(roundID, callerID string) error {
	round, err := uc.roundRepo.GetByID(roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return domain.ErrRoundNotFound
	}
	if round.PayerID != callerID {
		return domain.ErrRoundNotPayer
	}
	if round.Status != domain.RoundStatusPending {
		return domain.ErrRoundNotPending
	}
	round.Status = domain.RoundStatusOpen
	if err := uc.roundRepo.Update(round); err != nil {
		return err
	}
	// Notifica todos os membros da rotação que a rodada está aberta para participação
	rotation, err := uc.rotationRepo.Get()
	if err == nil && rotation != nil && len(rotation.Members) > 0 {
		userIDs := make([]string, len(rotation.Members))
		for i, m := range rotation.Members {
			userIDs[i] = m.UserID
		}
		_ = uc.notifySvc.SendParticipationOpen(userIDs)
	}
	return nil
}
```

- [ ] **Step 2: Verificar que todos os testes passam**

```bash
go test ./internal/usecase/... ./internal/handler/http/...
```

Esperado: todos os testes passam (os mocks já foram atualizados na Task 3)

- [ ] **Step 3: Commit**

```bash
git add internal/usecase/round.go
git commit -m "feat(participation): notify participants open on round confirm"
```

---

### Task 7: ParticipationHandler

**Files:**
- Create: `internal/handler/http/participation.go`

- [ ] **Step 1: Criar internal/handler/http/participation.go**

```go
package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

type ParticipationHandler struct {
	uc *usecase.ParticipationUseCase
}

func NewParticipationHandler(uc *usecase.ParticipationUseCase) *ParticipationHandler {
	return &ParticipationHandler{uc: uc}
}

type participateRequest struct {
	Quantity int `json:"quantity"`
}

// Participate godoc
// @Summary      Participar da rodada
// @Description  Usuário registra participação na rodada aberta com a quantidade desejada
// @Tags         participations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string              true  "Round ID"
// @Param        body  body      participateRequest  true  "Quantidade"
// @Success      201   {object}  map[string]string
// @Failure      400   {object}  ErrValidation
// @Failure      401   {object}  ErrInvalidCredentials
// @Failure      404   {object}  ErrValidation
// @Failure      409   {object}  ErrValidation  "round not open or already participating"
// @Router       /rounds/{id}/participate [post]
func (h *ParticipationHandler) Participate(w http.ResponseWriter, r *http.Request) {
	roundID := chi.URLParam(r, "id")
	userID := UserIDFromContext(r.Context())

	var req participateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Quantity < 1 {
		writeError(w, http.StatusBadRequest, "quantity is required and must be >= 1")
		return
	}

	if err := h.uc.Participate(roundID, userID, req.Quantity); err != nil {
		switch {
		case errors.Is(err, domain.ErrRoundNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrRoundNotOpen):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrAlreadyParticipating):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "participation registered"})
}

// Withdraw godoc
// @Summary      Retirar participação
// @Description  Usuário cancela sua participação na rodada aberta
// @Tags         participations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Round ID"
// @Success      204
// @Failure      401  {object}  ErrInvalidCredentials
// @Failure      404  {object}  ErrValidation
// @Failure      409  {object}  ErrValidation  "round not open"
// @Router       /rounds/{id}/participate [delete]
func (h *ParticipationHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	roundID := chi.URLParam(r, "id")
	userID := UserIDFromContext(r.Context())

	if err := h.uc.Withdraw(roundID, userID); err != nil {
		switch {
		case errors.Is(err, domain.ErrRoundNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrParticipationNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrRoundNotOpen):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetParticipations godoc
// @Summary      Listar participações
// @Description  Lista todos os participantes da rodada e o total de pães
// @Tags         participations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Round ID"
// @Success      200  {object}  usecase.ParticipationsResponse
// @Failure      401  {object}  ErrInvalidCredentials
// @Router       /rounds/{id}/participations [get]
func (h *ParticipationHandler) GetParticipations(w http.ResponseWriter, r *http.Request) {
	roundID := chi.URLParam(r, "id")

	resp, err := h.uc.GetParticipations(roundID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
```

- [ ] **Step 2: Verificar compilação**

```bash
go build ./...
```

Esperado: sem erros

- [ ] **Step 3: Commit**

```bash
git add internal/handler/http/participation.go
git commit -m "feat(participation): add ParticipationHandler with 3 endpoints and Swagger docs"
```

---

### Task 8: Wiring (main.go + api.go)

**Files:**
- Modify: `cmd/main.go`
- Modify: `cmd/api.go`

- [ ] **Step 1: Adicionar participationHandler ao struct application em cmd/api.go**

Localizar o struct `application` e adicionar o campo:

```go
type application struct {
	config               config
	userHandler          *handler.UserHandler
	authHandler          *handler.AuthHandler
	configHandler        *handler.ConfigHandler
	rotationHandler      *handler.RotationHandler
	roundHandler         *handler.RoundHandler
	participationHandler *handler.ParticipationHandler
}
```

- [ ] **Step 2: Registrar as 3 rotas em cmd/api.go**

Dentro do grupo autenticado, após as rotas de rounds, adicionar:

```go
r.Post("/rounds/{id}/participate", app.participationHandler.Participate)
r.Delete("/rounds/{id}/participate", app.participationHandler.Withdraw)
r.Get("/rounds/{id}/participations", app.participationHandler.GetParticipations)
```

- [ ] **Step 3: Criar repo, usecase e handler em cmd/main.go**

Após a linha `roundHandler := handler.NewRoundHandler(roundUC)`, adicionar:

```go
participationRepo    := postgres.NewParticipationRepository(gormDB)
participationUC      := usecase.NewParticipationUseCase(participationRepo, roundRepo)
participationHandler := handler.NewParticipationHandler(participationUC)
```

- [ ] **Step 4: Passar participationHandler ao application em cmd/main.go**

No bloco de criação do `api`, adicionar o campo:

```go
api := &application{
	config:               *cfg,
	userHandler:          userHandler,
	authHandler:          authHandler,
	configHandler:        configHandler,
	rotationHandler:      rotationHandler,
	roundHandler:         roundHandler,
	participationHandler: participationHandler,
}
```

- [ ] **Step 5: Verificar compilação final**

```bash
go build ./...
```

Esperado: sem erros

- [ ] **Step 6: Rodar todos os testes**

```bash
go test ./...
```

Esperado: todos os testes passam

- [ ] **Step 7: Commit**

```bash
git add cmd/main.go cmd/api.go
git commit -m "feat(participation): wire up ParticipationHandler and register routes"
```

---

## Verificação Final

- [ ] Compilação limpa: `go build ./...`
- [ ] Todos os testes: `go test ./...`
- [ ] Endpoints existentes de round não foram quebrados
- [ ] Branch pronta para PR contra `master`
