package score_test

import (
	"testing"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/score"
)

// --- mocks ---

type mockRoundRepo struct {
	rounds map[string]*domain.Round
}

func (m *mockRoundRepo) Create(r *domain.Round) error                           { return nil }
func (m *mockRoundRepo) GetByDate(date string) (*domain.Round, error)           { return nil, nil }
func (m *mockRoundRepo) GetByID(id string) (*domain.Round, error)               { return m.rounds[id], nil }
func (m *mockRoundRepo) GetAll(page, limit int) ([]*domain.Round, int64, error) { return nil, 0, nil }
func (m *mockRoundRepo) Update(r *domain.Round) error                           { return nil }

type mockPartRepo struct {
	byRound map[string][]*domain.Participation
}

func (m *mockPartRepo) Create(p *domain.Participation) error { return nil }
func (m *mockPartRepo) GetByRoundAndUser(roundID, userID string) (*domain.Participation, error) {
	return nil, nil
}
func (m *mockPartRepo) GetByRound(roundID string) ([]*domain.Participation, error) {
	return m.byRound[roundID], nil
}
func (m *mockPartRepo) Delete(roundID, userID string) error { return nil }

type mockConfigRepo struct {
	values map[string]string
}

func (m *mockConfigRepo) GetAll() ([]*domain.Config, error) {
	var out []*domain.Config
	for k, v := range m.values {
		out = append(out, &domain.Config{Key: k, Value: v})
	}
	return out, nil
}
func (m *mockConfigRepo) Set(key, value string) error { return nil }

type mockScoreRepo struct {
	scores   map[string]*domain.Score
	upserted []*domain.Score
}

func newMockScoreRepo() *mockScoreRepo {
	return &mockScoreRepo{scores: make(map[string]*domain.Score)}
}

func (m *mockScoreRepo) Upsert(s *domain.Score) error {
	m.upserted = append(m.upserted, s)
	clone := *s
	m.scores[s.UserID] = &clone
	return nil
}

func (m *mockScoreRepo) GetAll() ([]*domain.Score, error) {
	var out []*domain.Score
	for _, s := range m.scores {
		clone := *s
		out = append(out, &clone)
	}
	return out, nil
}

func (m *mockScoreRepo) GetByUserID(userID string) (*domain.Score, error) {
	s := m.scores[userID]
	if s == nil {
		return nil, nil
	}
	clone := *s
	return &clone, nil
}

// --- helpers ---

func closedRound(id, payerID string) *domain.Round {
	return &domain.Round{ID: id, PayerID: payerID, Status: domain.RoundStatusClosed}
}

func cancelledRound(id, payerID string) *domain.Round {
	return &domain.Round{ID: id, PayerID: payerID, Status: domain.RoundStatusCancelled}
}

func participation(roundID, userID string, qty int) *domain.Participation {
	return &domain.Participation{RoundID: roundID, UserID: userID, Quantity: qty}
}

// --- tests ---

func TestScoreUpdater_UpdateAfterRound_FirstPayment(t *testing.T) {
	roundRepo := &mockRoundRepo{rounds: map[string]*domain.Round{
		"r1": closedRound("r1", "payer-1"),
	}}
	partRepo := &mockPartRepo{byRound: map[string][]*domain.Participation{
		"r1": {participation("r1", "payer-1", 2), participation("r1", "user-2", 1)},
	}}
	configRepo := &mockConfigRepo{values: map[string]string{"price_per_unit": "5.00"}}
	scoreRepo := newMockScoreRepo()

	svc := score.NewScoreUpdater(roundRepo, partRepo, configRepo, scoreRepo)
	if err := svc.UpdateAfterRound("r1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// payer-1: times_paid=1, times_participated=1, total_amount_spent=(2+1)*5=15
	payerScore := scoreRepo.scores["payer-1"]
	if payerScore == nil {
		t.Fatal("expected score for payer-1")
	}
	if payerScore.TimesPaid != 1 {
		t.Errorf("expected TimesPaid=1, got %d", payerScore.TimesPaid)
	}
	if payerScore.TimesParticipated != 1 {
		t.Errorf("expected TimesParticipated=1, got %d", payerScore.TimesParticipated)
	}
	if payerScore.TotalAmountSpent != 15.0 {
		t.Errorf("expected TotalAmountSpent=15, got %f", payerScore.TotalAmountSpent)
	}

	// user-2: times_participated=1, streak=1
	user2Score := scoreRepo.scores["user-2"]
	if user2Score == nil {
		t.Fatal("expected score for user-2")
	}
	if user2Score.TimesParticipated != 1 {
		t.Errorf("expected TimesParticipated=1, got %d", user2Score.TimesParticipated)
	}
	if user2Score.CurrentStreak != 1 {
		t.Errorf("expected CurrentStreak=1, got %d", user2Score.CurrentStreak)
	}
}

func TestScoreUpdater_UpdateAfterRound_StreakReset(t *testing.T) {
	roundRepo := &mockRoundRepo{rounds: map[string]*domain.Round{
		"r2": closedRound("r2", "payer-1"),
	}}
	partRepo := &mockPartRepo{byRound: map[string][]*domain.Participation{
		"r2": {participation("r2", "payer-1", 1)},
	}}
	configRepo := &mockConfigRepo{values: map[string]string{"price_per_unit": "5.00"}}
	scoreRepo := newMockScoreRepo()
	// user-2 had a streak of 3 but did not participate in r2
	scoreRepo.scores["user-2"] = &domain.Score{UserID: "user-2", CurrentStreak: 3, TimesParticipated: 3}

	svc := score.NewScoreUpdater(roundRepo, partRepo, configRepo, scoreRepo)
	if err := svc.UpdateAfterRound("r2"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if scoreRepo.scores["user-2"].CurrentStreak != 0 {
		t.Errorf("expected streak reset to 0, got %d", scoreRepo.scores["user-2"].CurrentStreak)
	}
}

func TestScoreUpdater_UpdateAfterRound_ScoreFormula(t *testing.T) {
	roundRepo := &mockRoundRepo{rounds: map[string]*domain.Round{
		"r3": closedRound("r3", "payer-1"),
	}}
	partRepo := &mockPartRepo{byRound: map[string][]*domain.Participation{
		"r3": {participation("r3", "payer-1", 1)},
	}}
	configRepo := &mockConfigRepo{values: map[string]string{"price_per_unit": "10.00"}}
	scoreRepo := newMockScoreRepo()

	svc := score.NewScoreUpdater(roundRepo, partRepo, configRepo, scoreRepo)
	if err := svc.UpdateAfterRound("r3"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := scoreRepo.scores["payer-1"]
	if s == nil {
		t.Fatal("expected score for payer-1")
	}
	// ratio_pago_participado = 1/1 = 1.0 → 40
	// valor_gasto_normalizado = 10/10 = 1.0 → 30
	// taxa_ausencia = 0/(1+0) = 0 → 0
	// streak_bonus = min(1,10)/10 = 0.1 → 1
	// score = 40+30-0+1 = 71
	if s.Score != 71.0 {
		t.Errorf("expected score=71, got %f", s.Score)
	}
}

func TestScoreUpdater_UpdateOnCancel_IncrementsSkipCount(t *testing.T) {
	roundRepo := &mockRoundRepo{rounds: map[string]*domain.Round{
		"r4": cancelledRound("r4", "payer-1"),
	}}
	partRepo := &mockPartRepo{byRound: map[string][]*domain.Participation{}}
	configRepo := &mockConfigRepo{values: map[string]string{"price_per_unit": "5.00"}}
	scoreRepo := newMockScoreRepo()
	scoreRepo.scores["payer-1"] = &domain.Score{UserID: "payer-1", TimesPaid: 2, SkipCount: 0}

	svc := score.NewScoreUpdater(roundRepo, partRepo, configRepo, scoreRepo)
	if err := svc.UpdateOnCancel("r4"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if scoreRepo.scores["payer-1"].SkipCount != 1 {
		t.Errorf("expected SkipCount=1, got %d", scoreRepo.scores["payer-1"].SkipCount)
	}
}

func TestScoreUpdater_UpdateAfterRound_RoundNotFound(t *testing.T) {
	roundRepo := &mockRoundRepo{rounds: map[string]*domain.Round{}}
	partRepo := &mockPartRepo{}
	configRepo := &mockConfigRepo{values: map[string]string{"price_per_unit": "5.00"}}
	scoreRepo := newMockScoreRepo()

	svc := score.NewScoreUpdater(roundRepo, partRepo, configRepo, scoreRepo)
	err := svc.UpdateAfterRound("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent round, got nil")
	}
}
