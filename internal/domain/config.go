package domain

import "errors"

var (
	ErrConfigUnknownKey   = errors.New("unknown config key")
	ErrConfigInvalidValue = errors.New("invalid config value")
)

// ValidConfigKeys lista todas as chaves permitidas.
var ValidConfigKeys = map[string]bool{
	"notify_at":            true,
	"round_window_minutes": true,
	"price_per_unit":       true,
}

type Config struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ConfigRepository interface {
	GetAll() ([]*Config, error)
	Set(key, value string) error
}
