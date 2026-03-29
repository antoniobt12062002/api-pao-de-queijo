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

type stubConfigRepo struct {
	data map[string]string
}

func newStubConfigRepo() *stubConfigRepo {
	return &stubConfigRepo{data: map[string]string{
		"notify_at":            "08:00",
		"round_window_minutes": "30",
		"price_per_unit":       "2.50",
	}}
}
func (s *stubConfigRepo) GetAll() ([]*domain.Config, error) {
	configs := make([]*domain.Config, 0, len(s.data))
	for k, v := range s.data {
		configs = append(configs, &domain.Config{Key: k, Value: v})
	}
	return configs, nil
}
func (s *stubConfigRepo) Set(key, value string) error {
	s.data[key] = value
	return nil
}

func TestConfigHandler_GetAll(t *testing.T) {
	uc := usecase.NewConfigUseCase(newStubConfigRepo())
	h := handler.NewConfigHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/v1/config", nil)
	w := httptest.NewRecorder()
	h.GetAll(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp []map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 3 {
		t.Errorf("expected 3 configs, got %d", len(resp))
	}
}

func TestConfigHandler_Update_ValidKey(t *testing.T) {
	uc := usecase.NewConfigUseCase(newStubConfigRepo())
	h := handler.NewConfigHandler(uc)

	body := `{"key":"notify_at","value":"09:00"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/config", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestConfigHandler_Update_UnknownKey(t *testing.T) {
	uc := usecase.NewConfigUseCase(newStubConfigRepo())
	h := handler.NewConfigHandler(uc)

	body := `{"key":"hacker_key","value":"bad"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/config", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestConfigHandler_Update_InvalidValue(t *testing.T) {
	uc := usecase.NewConfigUseCase(newStubConfigRepo())
	h := handler.NewConfigHandler(uc)

	body := `{"key":"notify_at","value":"99:99"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/config", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestConfigHandler_Update_MissingBody(t *testing.T) {
	uc := usecase.NewConfigUseCase(newStubConfigRepo())
	h := handler.NewConfigHandler(uc)

	req := httptest.NewRequest(http.MethodPut, "/v1/config", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}
