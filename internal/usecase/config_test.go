package usecase_test

import (
	"errors"
	"testing"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

// mockConfigRepo implements domain.ConfigRepository for testing
type mockConfigRepo struct {
	data map[string]string
}

func newMockConfigRepo() *mockConfigRepo {
	return &mockConfigRepo{data: map[string]string{
		"notify_at":            "08:00",
		"round_window_minutes": "30",
		"price_per_unit":       "2.50",
	}}
}

func (m *mockConfigRepo) GetAll() ([]*domain.Config, error) {
	configs := make([]*domain.Config, 0, len(m.data))
	for k, v := range m.data {
		configs = append(configs, &domain.Config{Key: k, Value: v})
	}
	return configs, nil
}

func (m *mockConfigRepo) Set(key, value string) error {
	m.data[key] = value
	return nil
}

func TestConfigUseCase_GetAll(t *testing.T) {
	uc := usecase.NewConfigUseCase(newMockConfigRepo())
	configs, err := uc.GetAll()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(configs) != 3 {
		t.Errorf("expected 3 configs, got %d", len(configs))
	}
}

func TestConfigUseCase_Update_ValidKey(t *testing.T) {
	repo := newMockConfigRepo()
	uc := usecase.NewConfigUseCase(repo)
	err := uc.Update("notify_at", "09:00")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if repo.data["notify_at"] != "09:00" {
		t.Errorf("expected notify_at to be updated to 09:00, got: %s", repo.data["notify_at"])
	}
}

func TestConfigUseCase_Update_UnknownKey(t *testing.T) {
	uc := usecase.NewConfigUseCase(newMockConfigRepo())
	err := uc.Update("unknown_key", "value")
	if err == nil {
		t.Fatal("expected error for unknown key, got nil")
	}
	if !errors.Is(err, domain.ErrConfigUnknownKey) {
		t.Errorf("expected ErrConfigUnknownKey, got: %v", err)
	}
}

func TestConfigUseCase_Update_InvalidNotifyAt(t *testing.T) {
	uc := usecase.NewConfigUseCase(newMockConfigRepo())
	invalidValues := []string{
		"25:00",    // hora inválida
		"08:60",    // minuto inválido
		"8:00",     // sem zero à esquerda
		"08:00:00", // com segundos
		"",         // vazio
		"abc",      // não é hora
	}
	for _, v := range invalidValues {
		err := uc.Update("notify_at", v)
		if err == nil {
			t.Errorf("expected error for notify_at=%q, got nil", v)
		}
		if !errors.Is(err, domain.ErrConfigInvalidValue) {
			t.Errorf("expected ErrConfigInvalidValue for %q, got: %v", v, err)
		}
	}
}

func TestConfigUseCase_Update_InvalidWindowMinutes(t *testing.T) {
	uc := usecase.NewConfigUseCase(newMockConfigRepo())
	invalidValues := []string{"3", "241", "0", "-1", "abc", ""}
	for _, v := range invalidValues {
		err := uc.Update("round_window_minutes", v)
		if err == nil {
			t.Errorf("expected error for round_window_minutes=%q, got nil", v)
		}
		if !errors.Is(err, domain.ErrConfigInvalidValue) {
			t.Errorf("expected ErrConfigInvalidValue for %q, got: %v", v, err)
		}
	}
}

func TestConfigUseCase_Update_InvalidPricePerUnit(t *testing.T) {
	uc := usecase.NewConfigUseCase(newMockConfigRepo())
	invalidValues := []string{"-1", "0", "-0.01", "abc", ""}
	for _, v := range invalidValues {
		err := uc.Update("price_per_unit", v)
		if err == nil {
			t.Errorf("expected error for price_per_unit=%q, got nil", v)
		}
		if !errors.Is(err, domain.ErrConfigInvalidValue) {
			t.Errorf("expected ErrConfigInvalidValue for %q, got: %v", v, err)
		}
	}
}
