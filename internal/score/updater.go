package score

import (
	"fmt"
	"log/slog"
	"math"
	"strconv"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
)

// ScoreUpdaterImpl implements domain.ScoreUpdater.
type ScoreUpdaterImpl struct {
	roundRepo  domain.RoundRepository
	partRepo   domain.ParticipationRepository
	configRepo domain.ConfigRepository
	scoreRepo  domain.ScoreRepository
}

func NewScoreUpdater(
	roundRepo domain.RoundRepository,
	partRepo domain.ParticipationRepository,
	configRepo domain.ConfigRepository,
	scoreRepo domain.ScoreRepository,
) *ScoreUpdaterImpl {
	return &ScoreUpdaterImpl{
		roundRepo:  roundRepo,
		partRepo:   partRepo,
		configRepo: configRepo,
		scoreRepo:  scoreRepo,
	}
}

func (s *ScoreUpdaterImpl) UpdateAfterRound(roundID string) error {
	round, err := s.roundRepo.GetByID(roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return fmt.Errorf("ScoreUpdater: round %s not found", roundID)
	}

	parts, err := s.partRepo.GetByRound(roundID)
	if err != nil {
		return err
	}

	pricePerUnit, err := s.getPricePerUnit()
	if err != nil {
		slog.Warn("ScoreUpdater: price_per_unit not configured, using 0", "err", err)
		pricePerUnit = 0
	}

	// Build a map of existing scores for all known users.
	existing, err := s.scoreRepo.GetAll()
	if err != nil {
		return err
	}
	scoreMap := make(map[string]*domain.Score, len(existing))
	for _, sc := range existing {
		clone := *sc
		scoreMap[sc.UserID] = &clone
	}

	// Build participant set and compute total bill.
	participantSet := make(map[string]bool, len(parts))
	totalBill := 0.0
	for _, p := range parts {
		participantSet[p.UserID] = true
		totalBill += float64(p.Quantity) * pricePerUnit
	}

	// Update scores for participants.
	for _, p := range parts {
		sc, ok := scoreMap[p.UserID]
		if !ok {
			sc = &domain.Score{UserID: p.UserID}
			scoreMap[p.UserID] = sc
		}
		sc.TimesParticipated++
		sc.CurrentStreak++
	}

	// Update payer-specific fields.
	payerSc, ok := scoreMap[round.PayerID]
	if !ok {
		payerSc = &domain.Score{UserID: round.PayerID}
		scoreMap[round.PayerID] = payerSc
	}
	payerSc.TimesPaid++
	payerSc.TotalAmountSpent += totalBill

	// Reset streak for all users who did NOT participate.
	for userID, sc := range scoreMap {
		if !participantSet[userID] {
			sc.CurrentStreak = 0
		}
	}

	// Compute global max(total_amount_spent) for normalisation.
	var maxSpent float64
	for _, sc := range scoreMap {
		if sc.TotalAmountSpent > maxSpent {
			maxSpent = sc.TotalAmountSpent
		}
	}

	// Recalculate score for every user in the map and upsert.
	for _, sc := range scoreMap {
		sc.Score = calcScore(sc, maxSpent)
		if err := s.scoreRepo.Upsert(sc); err != nil {
			slog.Error("ScoreUpdater: error upserting score", "userID", sc.UserID, "err", err)
		}
	}

	return nil
}

func (s *ScoreUpdaterImpl) UpdateOnCancel(roundID string) error {
	round, err := s.roundRepo.GetByID(roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return fmt.Errorf("ScoreUpdater: round %s not found", roundID)
	}

	// Get or create score for payer.
	sc, err := s.scoreRepo.GetByUserID(round.PayerID)
	if err != nil {
		return err
	}
	if sc == nil {
		sc = &domain.Score{UserID: round.PayerID}
	}
	sc.SkipCount++

	// Recompute score using current global max.
	existing, err := s.scoreRepo.GetAll()
	if err != nil {
		return err
	}
	var maxSpent float64
	for _, e := range existing {
		if e.TotalAmountSpent > maxSpent {
			maxSpent = e.TotalAmountSpent
		}
	}
	if sc.TotalAmountSpent > maxSpent {
		maxSpent = sc.TotalAmountSpent
	}
	sc.Score = calcScore(sc, maxSpent)

	return s.scoreRepo.Upsert(sc)
}

func (s *ScoreUpdaterImpl) getPricePerUnit() (float64, error) {
	configs, err := s.configRepo.GetAll()
	if err != nil {
		return 0, err
	}
	for _, c := range configs {
		if c.Key == "price_per_unit" {
			return strconv.ParseFloat(c.Value, 64)
		}
	}
	return 0, fmt.Errorf("price_per_unit not found in config")
}

// calcScore computes the justice score for a user given the global max spent.
func calcScore(s *domain.Score, maxSpent float64) float64 {
	var ratioPagoParticipado float64
	if s.TimesParticipated > 0 {
		ratioPagoParticipado = float64(s.TimesPaid) / float64(s.TimesParticipated)
	}

	var valorGastoNormalizado float64
	if maxSpent > 0 {
		valorGastoNormalizado = s.TotalAmountSpent / maxSpent
	}

	var taxaAusencia float64
	denom := float64(s.TimesPaid + s.SkipCount)
	if denom > 0 {
		taxaAusencia = float64(s.SkipCount) / denom
	}

	streakBonus := math.Min(float64(s.CurrentStreak), 10) / 10

	raw := (ratioPagoParticipado * 40) +
		(valorGastoNormalizado * 30) -
		(taxaAusencia * 20) +
		(streakBonus * 10)

	return math.Max(0, raw)
}

// Verify interface compliance at compile time.
var _ domain.ScoreUpdater = (*ScoreUpdaterImpl)(nil)
