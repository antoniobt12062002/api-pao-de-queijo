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
