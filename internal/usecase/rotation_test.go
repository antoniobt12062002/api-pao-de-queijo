package usecase_test

import (
	"testing"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

// --- mocks ---

type mockRotationRepo struct {
	rotation *domain.Rotation
}

func newMockRotationRepo() *mockRotationRepo {
	return &mockRotationRepo{}
}

func (m *mockRotationRepo) Get() (*domain.Rotation, error) {
	return m.rotation, nil
}

func (m *mockRotationRepo) SetOrder(userIDs []string) error {
	members := make([]*domain.RotationMember, len(userIDs))
	for i, id := range userIDs {
		members[i] = &domain.RotationMember{UserID: id, Position: i}
	}
	if m.rotation == nil {
		m.rotation = &domain.Rotation{ID: "rotation-uuid", CurrentPos: 0}
	} else {
		m.rotation.CurrentPos = 0
	}
	m.rotation.Members = members
	return nil
}

func (m *mockRotationRepo) AdvancePosition() error {
	if m.rotation == nil || len(m.rotation.Members) == 0 {
		return domain.ErrRotationNotInitialized
	}
	m.rotation.CurrentPos = (m.rotation.CurrentPos + 1) % len(m.rotation.Members)
	return nil
}

type mockUserRepoForRotation struct {
	users []*domain.User
}

func newMockUserRepoForRotation(ids ...string) *mockUserRepoForRotation {
	users := make([]*domain.User, len(ids))
	for i, id := range ids {
		users[i] = &domain.User{ID: id, Name: "User " + id, Email: id + "@test.com"}
	}
	return &mockUserRepoForRotation{users: users}
}

func (m *mockUserRepoForRotation) Create(u *domain.User) error                              { return nil }
func (m *mockUserRepoForRotation) FindByEmail(email string) (*domain.User, error)           { return nil, nil }
func (m *mockUserRepoForRotation) FindByProviderID(p, id string) (*domain.User, error)      { return nil, nil }
func (m *mockUserRepoForRotation) FindByID(id string) (*domain.User, error)                 { return nil, nil }
func (m *mockUserRepoForRotation) UpdateRole(id, role string) error  { return nil }
func (m *mockUserRepoForRotation) Deactivate(id string) error        { return nil }
func (m *mockUserRepoForRotation) Activate(id string) error          { return nil }

func (m *mockUserRepoForRotation) FindAll() ([]*domain.User, error) {
	return m.users, nil
}

// --- tests ---

func TestRotationUseCase_GetCurrent_Empty(t *testing.T) {
	uc := usecase.NewRotationUseCase(newMockRotationRepo(), newMockUserRepoForRotation())
	resp, err := uc.GetCurrent()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(resp.Members) != 0 {
		t.Errorf("expected empty members, got %d", len(resp.Members))
	}
	if resp.CurrentPayerID != "" {
		t.Errorf("expected empty payer id, got %q", resp.CurrentPayerID)
	}
}

func TestRotationUseCase_GetCurrent_Initialized(t *testing.T) {
	repo := newMockRotationRepo()
	_ = repo.SetOrder([]string{"user-1", "user-2", "user-3"})
	repo.rotation.CurrentPos = 1

	uc := usecase.NewRotationUseCase(repo, newMockUserRepoForRotation("user-1", "user-2", "user-3"))
	resp, err := uc.GetCurrent()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.CurrentPayerID != "user-2" {
		t.Errorf("expected current payer to be user-2, got %q", resp.CurrentPayerID)
	}
	if len(resp.Members) != 3 {
		t.Errorf("expected 3 members, got %d", len(resp.Members))
	}
}

func TestRotationUseCase_UpdateOrder_Valid(t *testing.T) {
	repo := newMockRotationRepo()
	userRepo := newMockUserRepoForRotation("user-1", "user-2", "user-3")
	uc := usecase.NewRotationUseCase(repo, userRepo)

	err := uc.UpdateOrder([]string{"user-1", "user-2", "user-3"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(repo.rotation.Members) != 3 {
		t.Errorf("expected 3 members, got %d", len(repo.rotation.Members))
	}
	if repo.rotation.CurrentPos != 0 {
		t.Errorf("expected current_pos reset to 0, got %d", repo.rotation.CurrentPos)
	}
}

func TestRotationUseCase_UpdateOrder_Empty(t *testing.T) {
	uc := usecase.NewRotationUseCase(newMockRotationRepo(), newMockUserRepoForRotation())
	err := uc.UpdateOrder([]string{})
	if err == nil {
		t.Fatal("expected error for empty order, got nil")
	}
	if err != domain.ErrRotationEmptyOrder {
		t.Errorf("expected ErrRotationEmptyOrder, got: %v", err)
	}
}

func TestRotationUseCase_UpdateOrder_Duplicates(t *testing.T) {
	userRepo := newMockUserRepoForRotation("user-1", "user-2")
	uc := usecase.NewRotationUseCase(newMockRotationRepo(), userRepo)
	err := uc.UpdateOrder([]string{"user-1", "user-1"})
	if err == nil {
		t.Fatal("expected error for duplicate user, got nil")
	}
	if err != domain.ErrRotationDuplicateUser {
		t.Errorf("expected ErrRotationDuplicateUser, got: %v", err)
	}
}

func TestRotationUseCase_UpdateOrder_UnknownUser(t *testing.T) {
	userRepo := newMockUserRepoForRotation("user-1", "user-2")
	uc := usecase.NewRotationUseCase(newMockRotationRepo(), userRepo)
	err := uc.UpdateOrder([]string{"user-1", "unknown-user"})
	if err == nil {
		t.Fatal("expected error for unknown user, got nil")
	}
	if err != domain.ErrRotationUnknownUser {
		t.Errorf("expected ErrRotationUnknownUser, got: %v", err)
	}
}

func TestRotationUseCase_Skip_NotInitialized(t *testing.T) {
	uc := usecase.NewRotationUseCase(newMockRotationRepo(), newMockUserRepoForRotation())
	err := uc.Skip()
	if err == nil {
		t.Fatal("expected error for uninitialized rotation, got nil")
	}
	if err != domain.ErrRotationNotInitialized {
		t.Errorf("expected ErrRotationNotInitialized, got: %v", err)
	}
}

func TestRotationUseCase_Skip_AdvancesPosition(t *testing.T) {
	repo := newMockRotationRepo()
	_ = repo.SetOrder([]string{"user-1", "user-2", "user-3"})
	uc := usecase.NewRotationUseCase(repo, newMockUserRepoForRotation("user-1", "user-2", "user-3"))

	if err := uc.Skip(); err != nil {
		t.Fatalf("expected no error on first skip, got: %v", err)
	}
	if repo.rotation.CurrentPos != 1 {
		t.Errorf("expected pos 1 after first skip, got %d", repo.rotation.CurrentPos)
	}
}

func TestRotationUseCase_Skip_WrapsAround(t *testing.T) {
	repo := newMockRotationRepo()
	_ = repo.SetOrder([]string{"user-1", "user-2"})
	repo.rotation.CurrentPos = 1 // already at last position
	uc := usecase.NewRotationUseCase(repo, newMockUserRepoForRotation("user-1", "user-2"))

	if err := uc.Skip(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if repo.rotation.CurrentPos != 0 {
		t.Errorf("expected pos 0 after wrap, got %d", repo.rotation.CurrentPos)
	}
}
