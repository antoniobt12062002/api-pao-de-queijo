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

func (m *mockRepo) FindByID(id string) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) FindAll() ([]*domain.User, error) {
	users := make([]*domain.User, 0, len(m.users))
	for _, u := range m.users {
		users = append(users, u)
	}
	return users, nil
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
