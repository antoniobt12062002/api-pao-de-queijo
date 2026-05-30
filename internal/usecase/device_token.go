package usecase

import (
	"fmt"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
)

var validPlatforms = map[string]domain.DeviceTokenPlatform{
	"web":     domain.DevicePlatformWeb,
	"android": domain.DevicePlatformAndroid,
	"ios":     domain.DevicePlatformIOS,
}

type DeviceTokenUseCase struct {
	repo domain.DeviceTokenRepository
}

func NewDeviceTokenUseCase(repo domain.DeviceTokenRepository) *DeviceTokenUseCase {
	return &DeviceTokenUseCase{repo: repo}
}

// RegisterDevice upserts a device token for the given user.
// Returns an error if the platform is not one of: web, android, ios.
func (uc *DeviceTokenUseCase) RegisterDevice(userID, token, platform string) error {
	p, ok := validPlatforms[platform]
	if !ok {
		return fmt.Errorf("invalid platform %q: must be web, android, or ios", platform)
	}
	return uc.repo.Upsert(&domain.DeviceToken{
		UserID:   userID,
		Token:    token,
		Platform: p,
	})
}

// RemoveDevice deletes a device token.
// Returns ErrDeviceTokenNotFound if token doesn't exist.
// Returns ErrDeviceTokenForbidden if the token belongs to a different user.
func (uc *DeviceTokenUseCase) RemoveDevice(token, callerID string) error {
	dt, err := uc.repo.GetByToken(token)
	if err != nil {
		return err
	}
	if dt == nil {
		return domain.ErrDeviceTokenNotFound
	}
	if dt.UserID != callerID {
		return domain.ErrDeviceTokenForbidden
	}
	return uc.repo.DeleteByToken(token)
}
