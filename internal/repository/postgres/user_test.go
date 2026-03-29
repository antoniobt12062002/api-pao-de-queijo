package postgres_test

import (
	"os"
	"testing"

	"github.com/antoniobt12062002/pao-de-queijo/internal/db"
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	userrepo "github.com/antoniobt12062002/pao-de-queijo/internal/repository/postgres"
)

func setupDB(t *testing.T) *userrepo.UserRepository {
	t.Helper()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		t.Skip("DB_DSN not set, skipping integration test")
	}
	gormDB, err := db.New(dsn)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	return userrepo.NewUserRepository(gormDB)
}

func TestCreateAndFindByEmail(t *testing.T) {
	repo := setupDB(t)

	hash := "hashed"
	phone := "+5531999999999"
	user := &domain.User{
		Name:         "Test User",
		Email:        "test_repo@example.com",
		PasswordHash: &hash,
		Role:         "dev",
		Phone:        &phone,
		Provider:     "local",
	}

	err := repo.Create(user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if user.ID == "" {
		t.Fatal("expected ID to be set after create")
	}

	found, err := repo.FindByEmail("test_repo@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if found.Name != "Test User" {
		t.Errorf("expected name 'Test User', got '%s'", found.Name)
	}

	// cleanup
	repo.DB().Exec("DELETE FROM users WHERE email = ?", "test_repo@example.com")
}

func TestFindByProviderID(t *testing.T) {
	repo := setupDB(t)

	pid := "gh_123456"
	user := &domain.User{
		Name:       "GitHub User",
		Email:      "ghuser_repo@example.com",
		Role:       "dev",
		Provider:   "github",
		ProviderID: &pid,
	}

	err := repo.Create(user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := repo.FindByProviderID("github", "gh_123456")
	if err != nil {
		t.Fatalf("FindByProviderID failed: %v", err)
	}
	if found.Email != "ghuser_repo@example.com" {
		t.Errorf("expected email 'ghuser_repo@example.com', got '%s'", found.Email)
	}

	// cleanup
	repo.DB().Exec("DELETE FROM users WHERE email = ?", "ghuser_repo@example.com")
}
