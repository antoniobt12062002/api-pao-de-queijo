package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	handler "github.com/antoniobt12062002/pao-de-queijo/internal/handler/http"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

// --- stubs ---

type stubRoundRepo struct {
	rounds map[string]*domain.Round
	byDate map[string]*domain.Round
}

func newStubRoundRepo() *stubRoundRepo {
	return &stubRoundRepo{
		rounds: make(map[string]*domain.Round),
		byDate: make(map[string]*domain.Round),
	}
}

func (s *stubRoundRepo) Create(r *domain.Round) error {
	r.ID = "round-stub-1"
	s.rounds[r.ID] = r
	s.byDate[r.Date] = r
	return nil
}

func (s *stubRoundRepo) GetByDate(date string) (*domain.Round, error) {
	return s.byDate[date], nil
}

func (s *stubRoundRepo) GetByID(id string) (*domain.Round, error) {
	return s.rounds[id], nil
}

func (s *stubRoundRepo) GetAll(page, limit int) ([]*domain.Round, int64, error) {
	rounds := make([]*domain.Round, 0, len(s.rounds))
	for _, r := range s.rounds {
		rounds = append(rounds, r)
	}
	return rounds, int64(len(rounds)), nil
}

func (s *stubRoundRepo) Update(r *domain.Round) error {
	s.rounds[r.ID] = r
	s.byDate[r.Date] = r
	return nil
}

type stubRotationRepoForRound struct {
	rotation *domain.Rotation
}

func newStubRotationRepoForRound(members ...*domain.RotationMember) *stubRotationRepoForRound {
	var rot *domain.Rotation
	if len(members) > 0 {
		rot = &domain.Rotation{ID: "rot-1", CurrentPos: 0, Members: members}
	}
	return &stubRotationRepoForRound{rotation: rot}
}

func (s *stubRotationRepoForRound) Get() (*domain.Rotation, error) { return s.rotation, nil }
func (s *stubRotationRepoForRound) SetOrder(ids []string) error     { return nil }
func (s *stubRotationRepoForRound) AdvancePosition() error {
	if s.rotation != nil && len(s.rotation.Members) > 0 {
		s.rotation.CurrentPos = (s.rotation.CurrentPos + 1) % len(s.rotation.Members)
	}
	return nil
}

type stubNotifySvcForRound struct{}

func (s *stubNotifySvcForRound) SendRoundAnnounced(id string) error    { return nil }
func (s *stubNotifySvcForRound) SendRoundClosed(id string) error       { return nil }
func (s *stubNotifySvcForRound) SendReminder(ids []string) error        { return nil }
func (s *stubNotifySvcForRound) SendParticipationOpen(ids []string) error { return nil }

// withUserID injects user_id into context to simulate JWTMiddleware
func withUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), handler.ContextKeyUserID, userID)
	return r.WithContext(ctx)
}

// --- tests ---

func TestRoundHandler_GetAll_Empty(t *testing.T) {
	uc := usecase.NewRoundUseCase(newStubRoundRepo(), newStubRotationRepoForRound(), &stubNotifySvcForRound{})
	h := handler.NewRoundHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/v1/rounds", nil)
	w := httptest.NewRecorder()
	h.GetAll(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data, _ := resp["data"].([]any)
	if len(data) != 0 {
		t.Errorf("expected empty data, got %d", len(data))
	}
}

func TestRoundHandler_GetToday_NoRound(t *testing.T) {
	uc := usecase.NewRoundUseCase(newStubRoundRepo(), newStubRotationRepoForRound(), &stubNotifySvcForRound{})
	h := handler.NewRoundHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/v1/rounds/today", nil)
	req = withUserID(req, "user-1")
	w := httptest.NewRecorder()
	h.GetToday(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRoundHandler_GetToday_WithRound(t *testing.T) {
	repo := newStubRoundRepo()
	today := time.Now().Format("2006-01-02")
	round := &domain.Round{ID: "r1", Date: today, PayerID: "user-1", Status: domain.RoundStatusPending}
	repo.rounds["r1"] = round
	repo.byDate[today] = round

	uc := usecase.NewRoundUseCase(repo, newStubRotationRepoForRound(), &stubNotifySvcForRound{})
	h := handler.NewRoundHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/v1/rounds/today", nil)
	req = withUserID(req, "user-1")
	w := httptest.NewRecorder()
	h.GetToday(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["is_payer"] != true {
		t.Errorf("expected is_payer=true, got: %v", resp["is_payer"])
	}
}

func TestRoundHandler_Confirm_Valid(t *testing.T) {
	repo := newStubRoundRepo()
	round := &domain.Round{ID: "r1", Date: "2026-01-01", PayerID: "user-1", Status: domain.RoundStatusPending}
	repo.rounds["r1"] = round

	uc := usecase.NewRoundUseCase(repo, newStubRotationRepoForRound(), &stubNotifySvcForRound{})
	h := handler.NewRoundHandler(uc)

	r := chi.NewRouter()
	r.Post("/rounds/{id}/confirm", h.Confirm)

	req := httptest.NewRequest(http.MethodPost, "/rounds/r1/confirm", nil)
	req = withUserID(req, "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestRoundHandler_Confirm_NotPending(t *testing.T) {
	repo := newStubRoundRepo()
	round := &domain.Round{ID: "r1", Date: "2026-01-01", PayerID: "user-1", Status: domain.RoundStatusOpen}
	repo.rounds["r1"] = round

	uc := usecase.NewRoundUseCase(repo, newStubRotationRepoForRound(), &stubNotifySvcForRound{})
	h := handler.NewRoundHandler(uc)

	r := chi.NewRouter()
	r.Post("/rounds/{id}/confirm", h.Confirm)

	req := httptest.NewRequest(http.MethodPost, "/rounds/r1/confirm", nil)
	req = withUserID(req, "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestRoundHandler_Confirm_NotPayer(t *testing.T) {
	repo := newStubRoundRepo()
	round := &domain.Round{ID: "r1", Date: "2026-01-01", PayerID: "user-1", Status: domain.RoundStatusPending}
	repo.rounds["r1"] = round

	uc := usecase.NewRoundUseCase(repo, newStubRotationRepoForRound(), &stubNotifySvcForRound{})
	h := handler.NewRoundHandler(uc)

	r := chi.NewRouter()
	r.Post("/rounds/{id}/confirm", h.Confirm)

	req := httptest.NewRequest(http.MethodPost, "/rounds/r1/confirm", nil)
	req = withUserID(req, "user-2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRoundHandler_Cancel_Valid(t *testing.T) {
	repo := newStubRoundRepo()
	round := &domain.Round{ID: "r1", Date: "2026-01-01", PayerID: "user-1", Status: domain.RoundStatusPending}
	repo.rounds["r1"] = round
	repo.byDate["2026-01-01"] = round

	members := []*domain.RotationMember{
		{UserID: "user-1", Position: 0},
		{UserID: "user-2", Position: 1},
	}
	uc := usecase.NewRoundUseCase(repo, newStubRotationRepoForRound(members...), &stubNotifySvcForRound{})
	h := handler.NewRoundHandler(uc)

	r := chi.NewRouter()
	r.Post("/rounds/{id}/cancel", h.Cancel)

	req := httptest.NewRequest(http.MethodPost, "/rounds/r1/cancel", nil)
	req = withUserID(req, "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestRoundHandler_Cancel_NotPayer(t *testing.T) {
	repo := newStubRoundRepo()
	round := &domain.Round{ID: "r1", Date: "2026-01-01", PayerID: "user-1", Status: domain.RoundStatusPending}
	repo.rounds["r1"] = round

	uc := usecase.NewRoundUseCase(repo, newStubRotationRepoForRound(), &stubNotifySvcForRound{})
	h := handler.NewRoundHandler(uc)

	r := chi.NewRouter()
	r.Post("/rounds/{id}/cancel", h.Cancel)

	req := httptest.NewRequest(http.MethodPost, "/rounds/r1/cancel", nil)
	req = withUserID(req, "user-2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}
