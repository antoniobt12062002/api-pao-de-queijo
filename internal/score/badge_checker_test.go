package score_test

import (
	"testing"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/score"
)

// --- badge-specific mock ---

type mockBadgeRepo struct {
	inserted   []*domain.Badge
	topPayer   string
	bigSpender string
}

func (m *mockBadgeRepo) Insert(b *domain.Badge) error {
	m.inserted = append(m.inserted, b)
	return nil
}

func (m *mockBadgeRepo) GetByUserID(userID string) ([]*domain.Badge, error) { return nil, nil }

func (m *mockBadgeRepo) GetMonthlyTopRoundPayer(month string) (string, error) {
	return m.topPayer, nil
}

func (m *mockBadgeRepo) GetMonthlyBigSpender(month string, pricePerUnit float64) (string, error) {
	return m.bigSpender, nil
}

// --- helper ---

func hasBadge(inserted []*domain.Badge, userID string, badgeType domain.BadgeType) bool {
	for _, b := range inserted {
		if b.UserID == userID && b.Type == badgeType {
			return true
		}
	}
	return false
}

// --- tests ---

func TestBadgeChecker_NovoNaFila_AwardedOnFirstPayment(t *testing.T) {
	roundRepo := &mockRoundRepo{rounds: map[string]*domain.Round{
		"r1": closedRound("r1", "payer-1"),
	}}
	partRepo := &mockPartRepo{byRound: map[string][]*domain.Participation{
		"r1": {participation("r1", "payer-1", 1)},
	}}
	configRepo := &mockConfigRepo{values: map[string]string{"price_per_unit": "5.00"}}
	scoreRepo := newMockScoreRepo()
	scoreRepo.scores["payer-1"] = &domain.Score{UserID: "payer-1", TimesPaid: 1, SkipCount: 0}
	badgeRepo := &mockBadgeRepo{}

	checker := score.NewBadgeChecker(roundRepo, partRepo, configRepo, scoreRepo, badgeRepo)
	if err := checker.CheckAfterRound("r1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hasBadge(badgeRepo.inserted, "payer-1", domain.BadgeNovoNaFila) {
		t.Error("expected novo_na_fila badge for payer-1")
	}
}

func TestBadgeChecker_NuncaFoge_AwardedWhenNoSkips(t *testing.T) {
	roundRepo := &mockRoundRepo{rounds: map[string]*domain.Round{
		"r1": closedRound("r1", "payer-1"),
	}}
	partRepo := &mockPartRepo{byRound: map[string][]*domain.Participation{
		"r1": {participation("r1", "payer-1", 1)},
	}}
	configRepo := &mockConfigRepo{values: map[string]string{"price_per_unit": "5.00"}}
	scoreRepo := newMockScoreRepo()
	scoreRepo.scores["payer-1"] = &domain.Score{UserID: "payer-1", TimesPaid: 3, SkipCount: 0}
	badgeRepo := &mockBadgeRepo{}

	checker := score.NewBadgeChecker(roundRepo, partRepo, configRepo, scoreRepo, badgeRepo)
	if err := checker.CheckAfterRound("r1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hasBadge(badgeRepo.inserted, "payer-1", domain.BadgeNuncaFoge) {
		t.Error("expected nunca_foge badge for payer-1")
	}
}

func TestBadgeChecker_NuncaFoge_NotAwardedWithSkip(t *testing.T) {
	roundRepo := &mockRoundRepo{rounds: map[string]*domain.Round{
		"r1": closedRound("r1", "payer-1"),
	}}
	partRepo := &mockPartRepo{byRound: map[string][]*domain.Participation{
		"r1": {participation("r1", "payer-1", 1)},
	}}
	configRepo := &mockConfigRepo{values: map[string]string{"price_per_unit": "5.00"}}
	scoreRepo := newMockScoreRepo()
	scoreRepo.scores["payer-1"] = &domain.Score{UserID: "payer-1", TimesPaid: 3, SkipCount: 1}
	badgeRepo := &mockBadgeRepo{}

	checker := score.NewBadgeChecker(roundRepo, partRepo, configRepo, scoreRepo, badgeRepo)
	if err := checker.CheckAfterRound("r1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hasBadge(badgeRepo.inserted, "payer-1", domain.BadgeNuncaFoge) {
		t.Error("nunca_foge should NOT be awarded when SkipCount > 0")
	}
}

func TestBadgeChecker_QueijeiroFiel_AwardedOnStreak30(t *testing.T) {
	roundRepo := &mockRoundRepo{rounds: map[string]*domain.Round{
		"r1": closedRound("r1", "payer-1"),
	}}
	partRepo := &mockPartRepo{byRound: map[string][]*domain.Participation{
		"r1": {participation("r1", "user-2", 1)},
	}}
	configRepo := &mockConfigRepo{values: map[string]string{"price_per_unit": "5.00"}}
	scoreRepo := newMockScoreRepo()
	scoreRepo.scores["user-2"] = &domain.Score{UserID: "user-2", CurrentStreak: 30}
	badgeRepo := &mockBadgeRepo{}

	checker := score.NewBadgeChecker(roundRepo, partRepo, configRepo, scoreRepo, badgeRepo)
	if err := checker.CheckAfterRound("r1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hasBadge(badgeRepo.inserted, "user-2", domain.BadgeQueijeiroFiel) {
		t.Error("expected queijeiro_fiel badge for user-2 with streak 30")
	}
}

func TestBadgeChecker_PapaiNoel_AwardedToTopRoundPayer(t *testing.T) {
	roundRepo := &mockRoundRepo{rounds: map[string]*domain.Round{
		"r1": {ID: "r1", PayerID: "payer-1", Status: domain.RoundStatusClosed,
			Date: time.Now().Format("2006-01-02")},
	}}
	partRepo := &mockPartRepo{byRound: map[string][]*domain.Participation{
		"r1": {participation("r1", "payer-1", 1)},
	}}
	configRepo := &mockConfigRepo{values: map[string]string{"price_per_unit": "5.00"}}
	scoreRepo := newMockScoreRepo()
	scoreRepo.scores["payer-1"] = &domain.Score{UserID: "payer-1", TimesPaid: 1}
	badgeRepo := &mockBadgeRepo{topPayer: "payer-1"}

	checker := score.NewBadgeChecker(roundRepo, partRepo, configRepo, scoreRepo, badgeRepo)
	if err := checker.CheckAfterRound("r1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	month := time.Now().Format("2006-01")
	found := false
	for _, b := range badgeRepo.inserted {
		if b.UserID == "payer-1" && b.Type == domain.BadgePapaiNoel && b.Period == month {
			found = true
		}
	}
	if !found {
		t.Errorf("expected papai_noel badge for payer-1 this month (%s)", month)
	}
}

func TestBadgeChecker_BigSpender_AwardedToTopSpender(t *testing.T) {
	roundRepo := &mockRoundRepo{rounds: map[string]*domain.Round{
		"r1": {ID: "r1", PayerID: "payer-1", Status: domain.RoundStatusClosed,
			Date: time.Now().Format("2006-01-02")},
	}}
	partRepo := &mockPartRepo{byRound: map[string][]*domain.Participation{
		"r1": {participation("r1", "payer-1", 1)},
	}}
	configRepo := &mockConfigRepo{values: map[string]string{"price_per_unit": "5.00"}}
	scoreRepo := newMockScoreRepo()
	scoreRepo.scores["payer-1"] = &domain.Score{UserID: "payer-1", TimesPaid: 1}
	badgeRepo := &mockBadgeRepo{bigSpender: "payer-1"}

	checker := score.NewBadgeChecker(roundRepo, partRepo, configRepo, scoreRepo, badgeRepo)
	if err := checker.CheckAfterRound("r1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	month := time.Now().Format("2006-01")
	found := false
	for _, b := range badgeRepo.inserted {
		if b.UserID == "payer-1" && b.Type == domain.BadgeBigSpender && b.Period == month {
			found = true
		}
	}
	if !found {
		t.Errorf("expected big_spender badge for payer-1 this month (%s)", month)
	}
}
