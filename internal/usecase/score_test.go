package usecase_test

import (
	"testing"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

// --- mocks ---

type mockScoreRepoUC struct {
	all    []*domain.Score
	byUser map[string]*domain.Score
}

func (m *mockScoreRepoUC) Upsert(s *domain.Score) error { return nil }
func (m *mockScoreRepoUC) GetAll() ([]*domain.Score, error) { return m.all, nil }
func (m *mockScoreRepoUC) GetByUserID(userID string) (*domain.Score, error) {
	return m.byUser[userID], nil
}

type mockBadgeRepoUC struct {
	byUser map[string][]*domain.Badge
}

func (m *mockBadgeRepoUC) Insert(b *domain.Badge) error { return nil }
func (m *mockBadgeRepoUC) GetByUserID(userID string) ([]*domain.Badge, error) {
	return m.byUser[userID], nil
}
func (m *mockBadgeRepoUC) GetMonthlyTopRoundPayer(month string) (string, error) { return "", nil }
func (m *mockBadgeRepoUC) GetMonthlyBigSpender(month string, p float64) (string, error) {
	return "", nil
}

type mockUserRepoScore struct {
	byID map[string]*domain.User
}

func (m *mockUserRepoScore) Create(u *domain.User) error                         { return nil }
func (m *mockUserRepoScore) FindByEmail(email string) (*domain.User, error)      { return nil, nil }
func (m *mockUserRepoScore) FindByProviderID(p, id string) (*domain.User, error) { return nil, nil }
func (m *mockUserRepoScore) FindByID(id string) (*domain.User, error)            { return m.byID[id], nil }
func (m *mockUserRepoScore) FindAll() ([]*domain.User, error)                    { return nil, nil }
func (m *mockUserRepoScore) UpdateRole(id, role string) error                    { return nil }
func (m *mockUserRepoScore) Deactivate(id string) error                          { return nil }
func (m *mockUserRepoScore) Activate(id string) error                            { return nil }

// --- tests ---

func TestScoreUseCase_GetRanking_OrderedByScore(t *testing.T) {
	scoreRepo := &mockScoreRepoUC{
		all: []*domain.Score{
			{UserID: "u1", Score: 50},
			{UserID: "u2", Score: 80},
			{UserID: "u3", Score: 30},
		},
	}
	badgeRepo := &mockBadgeRepoUC{}
	userRepo := &mockUserRepoScore{
		byID: map[string]*domain.User{
			"u1": {ID: "u1", Name: "Alice", Email: "alice@example.com"},
			"u2": {ID: "u2", Name: "Bob", Email: "bob@example.com"},
			"u3": {ID: "u3", Name: "Carol", Email: "carol@example.com"},
		},
	}

	uc := usecase.NewScoreUseCase(scoreRepo, badgeRepo, userRepo)
	ranking, err := uc.GetRanking()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ranking) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(ranking))
	}
	if ranking[0].Score.Score != 80 {
		t.Errorf("expected first score=80, got %f", ranking[0].Score.Score)
	}
	if ranking[2].Score.Score != 30 {
		t.Errorf("expected last score=30, got %f", ranking[2].Score.Score)
	}
}

func TestScoreUseCase_GetUserScore_NotFound(t *testing.T) {
	scoreRepo := &mockScoreRepoUC{byUser: map[string]*domain.Score{}}
	badgeRepo := &mockBadgeRepoUC{}
	userRepo := &mockUserRepoScore{byID: map[string]*domain.User{}}

	uc := usecase.NewScoreUseCase(scoreRepo, badgeRepo, userRepo)
	_, err := uc.GetUserScore("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent user, got nil")
	}
}

func TestScoreUseCase_GetUserBadges_ReturnsBadges(t *testing.T) {
	scoreRepo := &mockScoreRepoUC{byUser: map[string]*domain.Score{"u1": {UserID: "u1"}}}
	badgeRepo := &mockBadgeRepoUC{
		byUser: map[string][]*domain.Badge{
			"u1": {
				{ID: "b1", UserID: "u1", Type: domain.BadgeNovoNaFila, EarnedAt: time.Now()},
			},
		},
	}
	userRepo := &mockUserRepoScore{byID: map[string]*domain.User{"u1": {ID: "u1", Name: "Alice"}}}

	uc := usecase.NewScoreUseCase(scoreRepo, badgeRepo, userRepo)
	badges, err := uc.GetUserBadges("u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(badges) != 1 {
		t.Errorf("expected 1 badge, got %d", len(badges))
	}
}
