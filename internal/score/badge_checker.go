package score

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
)

// BadgeCheckerImpl implements domain.BadgeChecker.
type BadgeCheckerImpl struct {
	roundRepo  domain.RoundRepository
	partRepo   domain.ParticipationRepository
	configRepo domain.ConfigRepository
	scoreRepo  domain.ScoreRepository
	badgeRepo  domain.BadgeRepository
}

func NewBadgeChecker(
	roundRepo domain.RoundRepository,
	partRepo domain.ParticipationRepository,
	configRepo domain.ConfigRepository,
	scoreRepo domain.ScoreRepository,
	badgeRepo domain.BadgeRepository,
) *BadgeCheckerImpl {
	return &BadgeCheckerImpl{
		roundRepo:  roundRepo,
		partRepo:   partRepo,
		configRepo: configRepo,
		scoreRepo:  scoreRepo,
		badgeRepo:  badgeRepo,
	}
}

func (c *BadgeCheckerImpl) CheckAfterRound(roundID string) error {
	round, err := c.roundRepo.GetByID(roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return fmt.Errorf("BadgeChecker: round %s not found", roundID)
	}

	parts, err := c.partRepo.GetByRound(roundID)
	if err != nil {
		return err
	}

	month := time.Now().Format("2006-01")

	// --- Permanent badges for payer ---
	if payerScore, err := c.scoreRepo.GetByUserID(round.PayerID); err == nil && payerScore != nil {
		if payerScore.TimesPaid == 1 {
			c.insertBadge(round.PayerID, domain.BadgeNovoNaFila, "")
		}
		if payerScore.SkipCount == 0 && payerScore.TimesPaid > 0 {
			c.insertBadge(round.PayerID, domain.BadgeNuncaFoge, "")
		}
	}

	// --- Permanent badge for all participants: queijeiro_fiel ---
	for _, p := range parts {
		sc, err := c.scoreRepo.GetByUserID(p.UserID)
		if err != nil || sc == nil {
			continue
		}
		if sc.CurrentStreak >= 30 {
			c.insertBadge(p.UserID, domain.BadgeQueijeiroFiel, "")
		}
	}

	// --- Monthly badge: papai_noel ---
	topPayer, err := c.badgeRepo.GetMonthlyTopRoundPayer(month)
	if err != nil {
		slog.Error("BadgeChecker: error getting monthly top round payer", "err", err)
	} else if topPayer != "" {
		c.insertBadge(topPayer, domain.BadgePapaiNoel, month)
	}

	// --- Monthly badge: big_spender ---
	pricePerUnit, _ := c.getPricePerUnit()
	bigSpender, err := c.badgeRepo.GetMonthlyBigSpender(month, pricePerUnit)
	if err != nil {
		slog.Error("BadgeChecker: error getting monthly big spender", "err", err)
	} else if bigSpender != "" {
		c.insertBadge(bigSpender, domain.BadgeBigSpender, month)
	}

	return nil
}

func (c *BadgeCheckerImpl) insertBadge(userID string, badgeType domain.BadgeType, period string) {
	b := &domain.Badge{UserID: userID, Type: badgeType, Period: period}
	if err := c.badgeRepo.Insert(b); err != nil {
		slog.Error("BadgeChecker: error inserting badge", "type", badgeType, "userID", userID, "err", err)
	}
}

func (c *BadgeCheckerImpl) getPricePerUnit() (float64, error) {
	configs, err := c.configRepo.GetAll()
	if err != nil {
		return 0, err
	}
	for _, cfg := range configs {
		if cfg.Key == "price_per_unit" {
			return strconv.ParseFloat(cfg.Value, 64)
		}
	}
	return 0, fmt.Errorf("price_per_unit not found in config")
}

// Verify interface compliance at compile time.
var _ domain.BadgeChecker = (*BadgeCheckerImpl)(nil)
