package usecase

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
)

var notifyAtPattern = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

type ConfigUseCase struct {
	repo domain.ConfigRepository
}

func NewConfigUseCase(repo domain.ConfigRepository) *ConfigUseCase {
	return &ConfigUseCase{repo: repo}
}

func (uc *ConfigUseCase) GetAll() ([]*domain.Config, error) {
	return uc.repo.GetAll()
}

func (uc *ConfigUseCase) Update(key, value string) error {
	if !domain.ValidConfigKeys[key] {
		return domain.ErrConfigUnknownKey
	}
	if err := validateConfigValue(key, value); err != nil {
		return err
	}
	return uc.repo.Set(key, value)
}

func validateConfigValue(key, value string) error {
	switch key {
	case "notify_at":
		if !notifyAtPattern.MatchString(value) {
			return fmt.Errorf("%w: notify_at must be HH:MM", domain.ErrConfigInvalidValue)
		}
	case "round_window_minutes":
		n, err := strconv.Atoi(value)
		if err != nil || n < 5 || n > 240 {
			return fmt.Errorf("%w: round_window_minutes must be between 5 and 240", domain.ErrConfigInvalidValue)
		}
	case "price_per_unit":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil || f <= 0 {
			return fmt.Errorf("%w: price_per_unit must be a positive number", domain.ErrConfigInvalidValue)
		}
	}
	return nil
}
