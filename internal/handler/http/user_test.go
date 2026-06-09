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
func (s *stubRepo) Create(u *domain.User) error {
	u.ID = "uuid-1"
	s.users[u.Email] = u
	return nil
}
func (s *stubRepo) FindByEmail(e string) (*domain.User, error) { u := s.users[e]; return u, nil }
func (s *stubRepo) FindByProviderID(p, id string) (*domain.User, error) { return nil, nil }
func (s *stubRepo) FindByID(id string) (*domain.User, error)            { return nil, nil }
func (s *stubRepo) FindAll() ([]*domain.User, error)         { return nil, nil }
func (s *stubRepo) UpdateRole(id, role string) error         { return nil }
func (s *stubRepo) Deactivate(id string) error               { return nil }

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
