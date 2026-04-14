package domain

import (
	"errors"
	"time"
)

type DeviceTokenPlatform string

const (
	DevicePlatformWeb     DeviceTokenPlatform = "web"
	DevicePlatformAndroid DeviceTokenPlatform = "android"
	DevicePlatformIOS     DeviceTokenPlatform = "ios"
)

var (
	ErrDeviceTokenNotFound  = errors.New("device token not found")
	ErrDeviceTokenForbidden = errors.New("device token belongs to another user")
)

type DeviceToken struct {
	ID        string
	UserID    string
	Token     string
	Platform  DeviceTokenPlatform
	CreatedAt time.Time
}

type DeviceTokenRepository interface {
	Upsert(dt *DeviceToken) error
	GetByToken(token string) (*DeviceToken, error)
	GetTokensByUserIDs(userIDs []string) ([]string, error)
	DeleteByToken(token string) error
}
