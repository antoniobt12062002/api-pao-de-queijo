package usecase_test

import (
	"testing"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

// --- mocks ---

type mockDeviceTokenRepoUC struct {
	tokens   map[string]*domain.DeviceToken // key: token string
	deleted  []string
	upserted []*domain.DeviceToken
}

func newMockDeviceTokenRepoUC() *mockDeviceTokenRepoUC {
	return &mockDeviceTokenRepoUC{tokens: make(map[string]*domain.DeviceToken)}
}

func (m *mockDeviceTokenRepoUC) Upsert(dt *domain.DeviceToken) error {
	m.upserted = append(m.upserted, dt)
	m.tokens[dt.Token] = dt
	return nil
}

func (m *mockDeviceTokenRepoUC) GetByToken(token string) (*domain.DeviceToken, error) {
	return m.tokens[token], nil
}

func (m *mockDeviceTokenRepoUC) GetTokensByUserIDs(userIDs []string) ([]string, error) {
	return nil, nil
}

func (m *mockDeviceTokenRepoUC) DeleteByToken(token string) error {
	m.deleted = append(m.deleted, token)
	delete(m.tokens, token)
	return nil
}

// --- tests ---

func TestDeviceTokenUseCase_RegisterDevice_OK(t *testing.T) {
	repo := newMockDeviceTokenRepoUC()
	uc := usecase.NewDeviceTokenUseCase(repo)

	err := uc.RegisterDevice("user-1", "fcm-token-xyz", "android")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(repo.upserted) != 1 {
		t.Errorf("expected 1 upsert call, got %d", len(repo.upserted))
	}
	if repo.upserted[0].UserID != "user-1" {
		t.Errorf("expected UserID user-1, got %s", repo.upserted[0].UserID)
	}
	if repo.upserted[0].Platform != domain.DevicePlatformAndroid {
		t.Errorf("expected platform android, got %s", repo.upserted[0].Platform)
	}
}

func TestDeviceTokenUseCase_RegisterDevice_InvalidPlatform(t *testing.T) {
	repo := newMockDeviceTokenRepoUC()
	uc := usecase.NewDeviceTokenUseCase(repo)

	err := uc.RegisterDevice("user-1", "fcm-token-xyz", "windows")
	if err == nil {
		t.Fatal("expected error for invalid platform, got nil")
	}
}

func TestDeviceTokenUseCase_RemoveDevice_OK(t *testing.T) {
	repo := newMockDeviceTokenRepoUC()
	repo.tokens["fcm-token-abc"] = &domain.DeviceToken{Token: "fcm-token-abc", UserID: "user-1"}
	uc := usecase.NewDeviceTokenUseCase(repo)

	err := uc.RemoveDevice("fcm-token-abc", "user-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != "fcm-token-abc" {
		t.Errorf("expected token to be deleted, deleted: %v", repo.deleted)
	}
}

func TestDeviceTokenUseCase_RemoveDevice_NotFound(t *testing.T) {
	repo := newMockDeviceTokenRepoUC()
	uc := usecase.NewDeviceTokenUseCase(repo)

	err := uc.RemoveDevice("nonexistent-token", "user-1")
	if err != domain.ErrDeviceTokenNotFound {
		t.Errorf("expected ErrDeviceTokenNotFound, got: %v", err)
	}
}

func TestDeviceTokenUseCase_RemoveDevice_Forbidden(t *testing.T) {
	repo := newMockDeviceTokenRepoUC()
	repo.tokens["fcm-token-abc"] = &domain.DeviceToken{Token: "fcm-token-abc", UserID: "user-1"}
	uc := usecase.NewDeviceTokenUseCase(repo)

	err := uc.RemoveDevice("fcm-token-abc", "user-2")
	if err != domain.ErrDeviceTokenForbidden {
		t.Errorf("expected ErrDeviceTokenForbidden, got: %v", err)
	}
}
