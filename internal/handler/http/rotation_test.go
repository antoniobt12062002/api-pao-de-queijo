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

// --- stubs ---

type stubRotationRepo struct {
	rotation *domain.Rotation
}

func newStubRotationRepo() *stubRotationRepo {
	return &stubRotationRepo{}
}

func (s *stubRotationRepo) Get() (*domain.Rotation, error) {
	return s.rotation, nil
}

func (s *stubRotationRepo) SetOrder(userIDs []string) error {
	members := make([]*domain.RotationMember, len(userIDs))
	for i, id := range userIDs {
		members[i] = &domain.RotationMember{UserID: id, Position: i}
	}
	if s.rotation == nil {
		s.rotation = &domain.Rotation{ID: "test-rotation", CurrentPos: 0}
	} else {
		s.rotation.CurrentPos = 0
	}
	s.rotation.Members = members
	return nil
}

func (s *stubRotationRepo) AdvancePosition() error {
	if s.rotation == nil || len(s.rotation.Members) == 0 {
		return domain.ErrRotationNotInitialized
	}
	s.rotation.CurrentPos = (s.rotation.CurrentPos + 1) % len(s.rotation.Members)
	return nil
}

type stubUserRepoForRotation struct {
	users []*domain.User
}

func newStubUserRepoForRotation(ids ...string) *stubUserRepoForRotation {
	users := make([]*domain.User, len(ids))
	for i, id := range ids {
		users[i] = &domain.User{ID: id, Name: "User", Email: id + "@test.com"}
	}
	return &stubUserRepoForRotation{users: users}
}

func (s *stubUserRepoForRotation) Create(u *domain.User) error                             { return nil }
func (s *stubUserRepoForRotation) FindByEmail(email string) (*domain.User, error)          { return nil, nil }
func (s *stubUserRepoForRotation) FindByProviderID(p, id string) (*domain.User, error)     { return nil, nil }
func (s *stubUserRepoForRotation) FindByID(id string) (*domain.User, error)                { return nil, nil }
func (s *stubUserRepoForRotation) FindAll() ([]*domain.User, error)     { return s.users, nil }
func (s *stubUserRepoForRotation) UpdateRole(id, role string) error     { return nil }
func (s *stubUserRepoForRotation) Deactivate(id string) error           { return nil }
func (s *stubUserRepoForRotation) Activate(id string) error             { return nil }

// --- tests ---

func TestRotationHandler_GetCurrent_Empty(t *testing.T) {
	uc := usecase.NewRotationUseCase(newStubRotationRepo(), newStubUserRepoForRotation())
	h := handler.NewRotationHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/v1/rotation", nil)
	w := httptest.NewRecorder()
	h.GetCurrent(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	members, _ := resp["members"].([]any)
	if len(members) != 0 {
		t.Errorf("expected empty members, got %d", len(members))
	}
}

func TestRotationHandler_UpdateOrder_Valid(t *testing.T) {
	userRepo := newStubUserRepoForRotation("user-1", "user-2")
	uc := usecase.NewRotationUseCase(newStubRotationRepo(), userRepo)
	h := handler.NewRotationHandler(uc)

	body := `{"user_ids":["user-1","user-2"]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/rotation/order", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UpdateOrder(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestRotationHandler_UpdateOrder_Empty(t *testing.T) {
	uc := usecase.NewRotationUseCase(newStubRotationRepo(), newStubUserRepoForRotation())
	h := handler.NewRotationHandler(uc)

	body := `{"user_ids":[]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/rotation/order", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UpdateOrder(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRotationHandler_UpdateOrder_Duplicates(t *testing.T) {
	userRepo := newStubUserRepoForRotation("user-1")
	uc := usecase.NewRotationUseCase(newStubRotationRepo(), userRepo)
	h := handler.NewRotationHandler(uc)

	body := `{"user_ids":["user-1","user-1"]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/rotation/order", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UpdateOrder(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRotationHandler_UpdateOrder_UnknownUser(t *testing.T) {
	userRepo := newStubUserRepoForRotation("user-1")
	uc := usecase.NewRotationUseCase(newStubRotationRepo(), userRepo)
	h := handler.NewRotationHandler(uc)

	body := `{"user_ids":["user-1","unknown-id"]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/rotation/order", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UpdateOrder(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRotationHandler_Skip_Valid(t *testing.T) {
	repo := newStubRotationRepo()
	_ = repo.SetOrder([]string{"user-1", "user-2"})
	uc := usecase.NewRotationUseCase(repo, newStubUserRepoForRotation("user-1", "user-2"))
	h := handler.NewRotationHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/v1/rotation/skip", nil)
	w := httptest.NewRecorder()
	h.Skip(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestRotationHandler_Skip_NotInitialized(t *testing.T) {
	uc := usecase.NewRotationUseCase(newStubRotationRepo(), newStubUserRepoForRotation())
	h := handler.NewRotationHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/v1/rotation/skip", nil)
	w := httptest.NewRecorder()
	h.Skip(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}
