package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	handler "github.com/antoniobt12062002/pao-de-queijo/internal/handler/http"
)

// --- stub use case ---

type stubDeviceTokenUC struct {
	registerErr error
	removeErr   error
}

func (s *stubDeviceTokenUC) RegisterDevice(userID, token, platform string) error {
	return s.registerErr
}

func (s *stubDeviceTokenUC) RemoveDevice(token, callerID string) error {
	return s.removeErr
}

// --- tests ---

func TestDeviceTokenHandler_Register_OK(t *testing.T) {
	h := handler.NewDeviceTokenHandler(&stubDeviceTokenUC{})

	body, _ := json.Marshal(map[string]string{"token": "fcm-abc", "platform": "android"})
	req := httptest.NewRequest(http.MethodPost, "/v1/devices", bytes.NewReader(body))
	req = withUserID(req, "user-1")
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestDeviceTokenHandler_Register_MissingToken(t *testing.T) {
	h := handler.NewDeviceTokenHandler(&stubDeviceTokenUC{})

	body, _ := json.Marshal(map[string]string{"platform": "android"})
	req := httptest.NewRequest(http.MethodPost, "/v1/devices", bytes.NewReader(body))
	req = withUserID(req, "user-1")
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDeviceTokenHandler_Register_InvalidPlatform(t *testing.T) {
	h := handler.NewDeviceTokenHandler(&stubDeviceTokenUC{registerErr: domain.ErrDeviceTokenForbidden})

	body, _ := json.Marshal(map[string]string{"token": "fcm-abc", "platform": "windows"})
	req := httptest.NewRequest(http.MethodPost, "/v1/devices", bytes.NewReader(body))
	req = withUserID(req, "user-1")
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDeviceTokenHandler_Remove_OK(t *testing.T) {
	h := handler.NewDeviceTokenHandler(&stubDeviceTokenUC{})

	r := chi.NewRouter()
	r.Delete("/v1/devices/{token}", h.Remove)

	req := httptest.NewRequest(http.MethodDelete, "/v1/devices/fcm-abc", nil)
	req = withUserID(req, "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestDeviceTokenHandler_Remove_NotFound(t *testing.T) {
	h := handler.NewDeviceTokenHandler(&stubDeviceTokenUC{removeErr: domain.ErrDeviceTokenNotFound})

	r := chi.NewRouter()
	r.Delete("/v1/devices/{token}", h.Remove)

	req := httptest.NewRequest(http.MethodDelete, "/v1/devices/unknown-token", nil)
	req = withUserID(req, "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeviceTokenHandler_Remove_Forbidden(t *testing.T) {
	h := handler.NewDeviceTokenHandler(&stubDeviceTokenUC{removeErr: domain.ErrDeviceTokenForbidden})

	r := chi.NewRouter()
	r.Delete("/v1/devices/{token}", h.Remove)

	req := httptest.NewRequest(http.MethodDelete, "/v1/devices/fcm-abc", nil)
	req = withUserID(req, "user-2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}
