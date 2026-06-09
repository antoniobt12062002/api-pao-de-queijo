package usecase

import (
	"errors"
	"sort"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
)

var ErrScoreNotFound = errors.New("score not found for user")

// ScoreResponse combines a Score with user identity fields.
type ScoreResponse struct {
	*domain.Score
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

type ScoreUseCase struct {
	scoreRepo domain.ScoreRepository
	badgeRepo domain.BadgeRepository
	userRepo  domain.UserRepository
}

func NewScoreUseCase(
	scoreRepo domain.ScoreRepository,
	badgeRepo domain.BadgeRepository,
	userRepo domain.UserRepository,
) *ScoreUseCase {
	return &ScoreUseCase{
		scoreRepo: scoreRepo,
		badgeRepo: badgeRepo,
		userRepo:  userRepo,
	}
}

func (uc *ScoreUseCase) GetRanking() ([]*ScoreResponse, error) {
	scores, err := uc.scoreRepo.GetAll()
	if err != nil {
		return nil, err
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	out := make([]*ScoreResponse, 0, len(scores))
	for _, s := range scores {
		user, err := uc.userRepo.FindByID(s.UserID)
		if err != nil || user == nil {
			continue
		}
		out = append(out, &ScoreResponse{Score: s, UserName: user.Name, UserEmail: user.Email})
	}
	return out, nil
}

func (uc *ScoreUseCase) GetUserScore(userID string) (*ScoreResponse, error) {
	user, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrScoreNotFound
	}

	s, err := uc.scoreRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		s = &domain.Score{UserID: userID}
	}
	return &ScoreResponse{Score: s, UserName: user.Name, UserEmail: user.Email}, nil
}

// JusticeEntry holds fairness metrics for one user.
type JusticeEntry struct {
	UserID           string  `json:"user_id"`
	UserName         string  `json:"user_name"`
	TimesPaid        int     `json:"times_paid"`
	TimesParticipated int    `json:"times_participated"`
	ExpectedPays     float64 `json:"expected_pays"`
	Balance          float64 `json:"balance"`
	FairnessRatio    float64 `json:"fairness_ratio"`
}

// GetJusticeChart returns per-user fairness metrics based on participation
// proportionality: expected_pays = (user_participations / total_participations) * total_paid_rounds.
func (uc *ScoreUseCase) GetJusticeChart() ([]*JusticeEntry, error) {
	scores, err := uc.scoreRepo.GetAll()
	if err != nil {
		return nil, err
	}

	var totalPaid, totalParticipated int
	for _, s := range scores {
		totalPaid += s.TimesPaid
		totalParticipated += s.TimesParticipated
	}

	out := make([]*JusticeEntry, 0, len(scores))
	for _, s := range scores {
		user, err := uc.userRepo.FindByID(s.UserID)
		if err != nil || user == nil {
			continue
		}

		var expected, ratio float64
		if totalParticipated > 0 {
			expected = float64(s.TimesParticipated) / float64(totalParticipated) * float64(totalPaid)
		}
		if expected > 0 {
			ratio = float64(s.TimesPaid) / expected
		}

		out = append(out, &JusticeEntry{
			UserID:            s.UserID,
			UserName:          user.Name,
			TimesPaid:         s.TimesPaid,
			TimesParticipated: s.TimesParticipated,
			ExpectedPays:      expected,
			Balance:           float64(s.TimesPaid) - expected,
			FairnessRatio:     ratio,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Balance > out[j].Balance
	})

	return out, nil
}

func (uc *ScoreUseCase) GetUserBadges(userID string) ([]*domain.Badge, error) {
	user, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrScoreNotFound
	}
	return uc.badgeRepo.GetByUserID(userID)
}
