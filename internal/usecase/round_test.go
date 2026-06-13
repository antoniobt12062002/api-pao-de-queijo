package usecase_test

import (
	"testing"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

// --- mocks ---

type mockRoundRepo struct {
	rounds map[string]*domain.Round
	byDate map[string]*domain.Round
}

func newMockRoundRepo() *mockRoundRepo {
	return &mockRoundRepo{
		rounds: make(map[string]*domain.Round),
		byDate: make(map[string]*domain.Round),
	}
}

func (m *mockRoundRepo) Create(r *domain.Round) error {
	r.ID = "round-uuid-1"
	r.CreatedAt = time.Now()
	m.rounds[r.ID] = r
	m.byDate[r.Date] = r
	return nil
}

func (m *mockRoundRepo) GetByDate(date string) (*domain.Round, error) {
	return m.byDate[date], nil
}

func (m *mockRoundRepo) GetByID(id string) (*domain.Round, error) {
	return m.rounds[id], nil
}

func (m *mockRoundRepo) GetAll(page, limit int) ([]*domain.Round, int64, error) {
	rounds := make([]*domain.Round, 0, len(m.rounds))
	for _, r := range m.rounds {
		rounds = append(rounds, r)
	}
	return rounds, int64(len(rounds)), nil
}

func (m *mockRoundRepo) Update(r *domain.Round) error {
	m.rounds[r.ID] = r
	m.byDate[r.Date] = r
	return nil
}

type mockRotationRepoForRound struct {
	rotation *domain.Rotation
}

func newMockRotationRepoForRound(members ...*domain.RotationMember) *mockRotationRepoForRound {
	var rotation *domain.Rotation
	if len(members) > 0 {
		rotation = &domain.Rotation{ID: "rot-1", CurrentPos: 0, Members: members}
	}
	return &mockRotationRepoForRound{rotation: rotation}
}

func (m *mockRotationRepoForRound) Get() (*domain.Rotation, error) {
	return m.rotation, nil
}

func (m *mockRotationRepoForRound) SetOrder(userIDs []string) error { return nil }

func (m *mockRotationRepoForRound) AdvancePosition() error {
	if m.rotation == nil || len(m.rotation.Members) == 0 {
		return domain.ErrRotationNotInitialized
	}
	m.rotation.CurrentPos = (m.rotation.CurrentPos + 1) % len(m.rotation.Members)
	return nil
}

type mockNotifySvc struct{}

func (n *mockNotifySvc) SendRoundAnnounced(payerID, roundID string) error            { return nil }
func (n *mockNotifySvc) SendParticipationOpen(userIDs []string, roundID string) error { return nil }
func (n *mockNotifySvc) SendRoundClosed(payerID, roundID string) error               { return nil }
func (n *mockNotifySvc) SendReminder(participantIDs []string, roundID string) error  { return nil }
func (n *mockNotifySvc) SendManual(userIDs []string, title, body string) error       { return nil }

type mockScoreUpdater struct{}

func (m *mockScoreUpdater) UpdateAfterRound(roundID string) error { return nil }
func (m *mockScoreUpdater) UpdateOnCancel(roundID string) error   { return nil }


// --- tests ---

func TestRoundUseCase_GetAll_Empty(t *testing.T) {
	uc := usecase.NewRoundUseCase(newMockRoundRepo(), newMockRotationRepoForRound(), &mockNotifySvc{}, &mockScoreUpdater{})
	resp, err := uc.GetAll(1, 20)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("expected empty data, got %d", len(resp.Data))
	}
	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}
}

func TestRoundUseCase_GetToday_NoRound(t *testing.T) {
	uc := usecase.NewRoundUseCase(newMockRoundRepo(), newMockRotationRepoForRound(), &mockNotifySvc{}, &mockScoreUpdater{})
	resp, err := uc.GetToday("user-1", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp != nil {
		t.Errorf("expected nil response when no round, got: %+v", resp)
	}
}

func TestRoundUseCase_GetToday_IsPayer(t *testing.T) {
	repo := newMockRoundRepo()
	today := time.Now().Format("2006-01-02")
	round := &domain.Round{
		ID:      "round-1",
		Date:    today,
		PayerID: "user-1",
		Status:  domain.RoundStatusPending,
	}
	repo.rounds["round-1"] = round
	repo.byDate[today] = round

	uc := usecase.NewRoundUseCase(repo, newMockRotationRepoForRound(), &mockNotifySvc{}, &mockScoreUpdater{})
	resp, err := uc.GetToday("user-1", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected round, got nil")
	}
	if !resp.IsPayer {
		t.Error("expected is_payer=true for the payer")
	}
}

func TestRoundUseCase_GetToday_IsNotPayer(t *testing.T) {
	repo := newMockRoundRepo()
	today := time.Now().Format("2006-01-02")
	round := &domain.Round{
		ID:      "round-1",
		Date:    today,
		PayerID: "user-1",
		Status:  domain.RoundStatusPending,
	}
	repo.rounds["round-1"] = round
	repo.byDate[today] = round

	uc := usecase.NewRoundUseCase(repo, newMockRotationRepoForRound(), &mockNotifySvc{}, &mockScoreUpdater{})
	resp, err := uc.GetToday("user-2", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected round, got nil")
	}
	if resp.IsPayer {
		t.Error("expected is_payer=false for a non-payer")
	}
}

func TestRoundUseCase_Confirm_Valid(t *testing.T) {
	repo := newMockRoundRepo()
	round := &domain.Round{ID: "round-1", Date: "2026-01-01", PayerID: "user-1", Status: domain.RoundStatusPending, ClosesAt: time.Now().Add(time.Hour)}
	repo.rounds["round-1"] = round

	uc := usecase.NewRoundUseCase(repo, newMockRotationRepoForRound(), &mockNotifySvc{}, &mockScoreUpdater{})
	if err := uc.Confirm("round-1", "user-1"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if repo.rounds["round-1"].Status != domain.RoundStatusOpen {
		t.Errorf("expected status open, got %s", repo.rounds["round-1"].Status)
	}
}

func TestRoundUseCase_Confirm_NotPending(t *testing.T) {
	repo := newMockRoundRepo()
	round := &domain.Round{ID: "round-1", Date: "2026-01-01", PayerID: "user-1", Status: domain.RoundStatusOpen}
	repo.rounds["round-1"] = round

	uc := usecase.NewRoundUseCase(repo, newMockRotationRepoForRound(), &mockNotifySvc{}, &mockScoreUpdater{})
	err := uc.Confirm("round-1", "user-1")
	if err == nil {
		t.Fatal("expected error for non-pending round, got nil")
	}
	if err != domain.ErrRoundNotPending {
		t.Errorf("expected ErrRoundNotPending, got: %v", err)
	}
}

func TestRoundUseCase_Confirm_NotPayer(t *testing.T) {
	repo := newMockRoundRepo()
	round := &domain.Round{ID: "round-1", Date: "2026-01-01", PayerID: "user-1", Status: domain.RoundStatusPending}
	repo.rounds["round-1"] = round

	uc := usecase.NewRoundUseCase(repo, newMockRotationRepoForRound(), &mockNotifySvc{}, &mockScoreUpdater{})
	err := uc.Confirm("round-1", "user-2")
	if err == nil {
		t.Fatal("expected error for non-payer, got nil")
	}
	if err != domain.ErrRoundNotPayer {
		t.Errorf("expected ErrRoundNotPayer, got: %v", err)
	}
}

func TestRoundUseCase_Cancel_Valid(t *testing.T) {
	repo := newMockRoundRepo()
	round := &domain.Round{ID: "round-1", Date: "2026-01-01", PayerID: "user-1", Status: domain.RoundStatusPending}
	repo.rounds["round-1"] = round
	repo.byDate["2026-01-01"] = round

	members := []*domain.RotationMember{
		{UserID: "user-1", Position: 0},
		{UserID: "user-2", Position: 1},
	}
	rotRepo := newMockRotationRepoForRound(members...)

	uc := usecase.NewRoundUseCase(repo, rotRepo, &mockNotifySvc{}, &mockScoreUpdater{})
	if err := uc.Cancel("round-1", "user-1"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	updated := repo.rounds["round-1"]
	if updated.PayerID != "user-2" {
		t.Errorf("expected new payer to be user-2, got %s", updated.PayerID)
	}
	if updated.Status != domain.RoundStatusPending {
		t.Errorf("expected status pending after reassignment, got %s", updated.Status)
	}
}

func TestRoundUseCase_Cancel_NotPending(t *testing.T) {
	repo := newMockRoundRepo()
	round := &domain.Round{ID: "round-1", Date: "2026-01-01", PayerID: "user-1", Status: domain.RoundStatusOpen}
	repo.rounds["round-1"] = round

	uc := usecase.NewRoundUseCase(repo, newMockRotationRepoForRound(), &mockNotifySvc{}, &mockScoreUpdater{})
	err := uc.Cancel("round-1", "user-1")
	if err == nil {
		t.Fatal("expected error for non-pending round, got nil")
	}
	if err != domain.ErrRoundNotPending {
		t.Errorf("expected ErrRoundNotPending, got: %v", err)
	}
}

func TestRoundUseCase_Cancel_NotPayer(t *testing.T) {
	repo := newMockRoundRepo()
	round := &domain.Round{ID: "round-1", Date: "2026-01-01", PayerID: "user-1", Status: domain.RoundStatusPending}
	repo.rounds["round-1"] = round

	uc := usecase.NewRoundUseCase(repo, newMockRotationRepoForRound(), &mockNotifySvc{}, &mockScoreUpdater{})
	err := uc.Cancel("round-1", "user-2")
	if err == nil {
		t.Fatal("expected error for non-payer, got nil")
	}
	if err != domain.ErrRoundNotPayer {
		t.Errorf("expected ErrRoundNotPayer, got: %v", err)
	}
}
