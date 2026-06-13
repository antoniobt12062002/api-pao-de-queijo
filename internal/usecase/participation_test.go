package usecase_test

import (
	"testing"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

// --- mocks ---

type mockUserRepo struct{}

func (m *mockUserRepo) Create(u *domain.User) error                                    { return nil }
func (m *mockUserRepo) FindByEmail(email string) (*domain.User, error)                  { return nil, nil }
func (m *mockUserRepo) FindByProviderID(p, id string) (*domain.User, error)             { return nil, nil }
func (m *mockUserRepo) FindByID(id string) (*domain.User, error)                        { return nil, nil }
func (m *mockUserRepo) FindAll() ([]*domain.User, error)  { return nil, nil }
func (m *mockUserRepo) UpdateRole(id, role string) error  { return nil }
func (m *mockUserRepo) Deactivate(id string) error        { return nil }
func (m *mockUserRepo) Activate(id string) error          { return nil }

type mockParticipationRepo struct {
	byRoundUser map[string]*domain.Participation // key: roundID+":"+userID
	byRound     map[string][]*domain.Participation
}

func newMockParticipationRepo() *mockParticipationRepo {
	return &mockParticipationRepo{
		byRoundUser: make(map[string]*domain.Participation),
		byRound:     make(map[string][]*domain.Participation),
	}
}

func (m *mockParticipationRepo) Create(p *domain.Participation) error {
	key := p.RoundID + ":" + p.UserID
	if _, exists := m.byRoundUser[key]; exists {
		return domain.ErrAlreadyParticipating
	}
	p.ID = "part-uuid-1"
	p.ConfirmedAt = time.Now()
	m.byRoundUser[key] = p
	m.byRound[p.RoundID] = append(m.byRound[p.RoundID], p)
	return nil
}

func (m *mockParticipationRepo) GetByRoundAndUser(roundID, userID string) (*domain.Participation, error) {
	key := roundID + ":" + userID
	return m.byRoundUser[key], nil
}

func (m *mockParticipationRepo) GetByRound(roundID string) ([]*domain.Participation, error) {
	return m.byRound[roundID], nil
}

func (m *mockParticipationRepo) Delete(roundID, userID string) error {
	key := roundID + ":" + userID
	if _, exists := m.byRoundUser[key]; !exists {
		return domain.ErrParticipationNotFound
	}
	delete(m.byRoundUser, key)
	list := m.byRound[roundID]
	for i, item := range list {
		if item.UserID == userID {
			m.byRound[roundID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	return nil
}

func openRound(id string) *domain.Round {
	return &domain.Round{ID: id, Date: "2026-01-01", PayerID: "payer-1", Status: domain.RoundStatusOpen}
}

func pendingRound(id string) *domain.Round {
	return &domain.Round{ID: id, Date: "2026-01-01", PayerID: "payer-1", Status: domain.RoundStatusPending}
}

// --- tests ---

func TestParticipationUseCase_Participate_OK(t *testing.T) {
	roundRepo := newMockRoundRepo()
	round := openRound("round-1")
	roundRepo.rounds["round-1"] = round

	uc := usecase.NewParticipationUseCase(newMockParticipationRepo(), roundRepo, &mockUserRepo{})
	if err := uc.Participate("round-1", "user-1", 2); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestParticipationUseCase_Participate_RoundNotFound(t *testing.T) {
	uc := usecase.NewParticipationUseCase(newMockParticipationRepo(), newMockRoundRepo(), &mockUserRepo{})
	err := uc.Participate("nonexistent", "user-1", 1)
	if err != domain.ErrRoundNotFound {
		t.Errorf("expected ErrRoundNotFound, got: %v", err)
	}
}

func TestParticipationUseCase_Participate_RoundNotOpen(t *testing.T) {
	roundRepo := newMockRoundRepo()
	roundRepo.rounds["round-1"] = pendingRound("round-1")

	uc := usecase.NewParticipationUseCase(newMockParticipationRepo(), roundRepo, &mockUserRepo{})
	err := uc.Participate("round-1", "user-1", 1)
	if err != domain.ErrRoundNotOpen {
		t.Errorf("expected ErrRoundNotOpen, got: %v", err)
	}
}

func TestParticipationUseCase_Participate_AlreadyParticipating(t *testing.T) {
	roundRepo := newMockRoundRepo()
	roundRepo.rounds["round-1"] = openRound("round-1")
	partRepo := newMockParticipationRepo()

	uc := usecase.NewParticipationUseCase(partRepo, roundRepo, &mockUserRepo{})
	_ = uc.Participate("round-1", "user-1", 1)
	err := uc.Participate("round-1", "user-1", 1)
	if err != domain.ErrAlreadyParticipating {
		t.Errorf("expected ErrAlreadyParticipating, got: %v", err)
	}
}

func TestParticipationUseCase_Withdraw_OK(t *testing.T) {
	roundRepo := newMockRoundRepo()
	roundRepo.rounds["round-1"] = openRound("round-1")
	partRepo := newMockParticipationRepo()

	uc := usecase.NewParticipationUseCase(partRepo, roundRepo, &mockUserRepo{})
	_ = uc.Participate("round-1", "user-1", 1)
	if err := uc.Withdraw("round-1", "user-1"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestParticipationUseCase_Withdraw_RoundNotOpen(t *testing.T) {
	roundRepo := newMockRoundRepo()
	roundRepo.rounds["round-1"] = pendingRound("round-1")

	uc := usecase.NewParticipationUseCase(newMockParticipationRepo(), roundRepo, &mockUserRepo{})
	err := uc.Withdraw("round-1", "user-1")
	if err != domain.ErrRoundNotOpen {
		t.Errorf("expected ErrRoundNotOpen, got: %v", err)
	}
}

func TestParticipationUseCase_Withdraw_NotFound(t *testing.T) {
	roundRepo := newMockRoundRepo()
	roundRepo.rounds["round-1"] = openRound("round-1")

	uc := usecase.NewParticipationUseCase(newMockParticipationRepo(), roundRepo, &mockUserRepo{})
	err := uc.Withdraw("round-1", "user-1")
	if err != domain.ErrParticipationNotFound {
		t.Errorf("expected ErrParticipationNotFound, got: %v", err)
	}
}

func TestParticipationUseCase_GetParticipations_Empty(t *testing.T) {
	uc := usecase.NewParticipationUseCase(newMockParticipationRepo(), newMockRoundRepo(), &mockUserRepo{})
	resp, err := uc.GetParticipations("round-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(resp.Participations) != 0 {
		t.Errorf("expected empty data, got %d", len(resp.Participations))
	}
	if resp.TotalQuantity != 0 {
		t.Errorf("expected total_quantity 0, got %d", resp.TotalQuantity)
	}
}

func TestParticipationUseCase_GetParticipations_WithData(t *testing.T) {
	roundRepo := newMockRoundRepo()
	roundRepo.rounds["round-1"] = openRound("round-1")
	partRepo := newMockParticipationRepo()

	uc := usecase.NewParticipationUseCase(partRepo, roundRepo, &mockUserRepo{})
	_ = uc.Participate("round-1", "user-1", 3)
	_ = uc.Participate("round-1", "user-2", 2)

	resp, err := uc.GetParticipations("round-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(resp.Participations) != 2 {
		t.Errorf("expected 2 participations, got %d", len(resp.Participations))
	}
	if resp.TotalQuantity != 5 {
		t.Errorf("expected total_quantity 5, got %d", resp.TotalQuantity)
	}
}
