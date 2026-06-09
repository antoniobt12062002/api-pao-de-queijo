package usecase

import (
	"errors"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists       = errors.New("email already registered")
	ErrEmailTakenByLocalAccount = errors.New("email already registered with password login")
	ErrInvalidCredentials       = errors.New("invalid email or password")
	ErrUserInactive             = errors.New("account is deactivated")
)

type UserUseCase struct {
	repo      domain.UserRepository
	jwtSecret string
}

func NewUserUseCase(repo domain.UserRepository, jwtSecret string) *UserUseCase {
	return &UserUseCase{repo: repo, jwtSecret: jwtSecret}
}

func (uc *UserUseCase) ListUsers() ([]*domain.User, error) {
	return uc.repo.FindAll()
}

func (uc *UserUseCase) UpdateRole(id, role string) error {
	if role != "admin" && role != "dev" {
		return errors.New("invalid role: must be admin or dev")
	}
	return uc.repo.UpdateRole(id, role)
}

func (uc *UserUseCase) DeactivateUser(id string) error {
	return uc.repo.Deactivate(id)
}

func (uc *UserUseCase) CreateUser(input domain.CreateUserInput) (*domain.User, error) {
	existing, err := uc.repo.FindByEmail(input.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, err
	}

	role := input.Role
	if role == "" {
		role = "dev"
	}

	hashStr := string(hash)
	user := &domain.User{
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: &hashStr,
		Role:         role,
		Phone:        input.Phone,
		Provider:     "local",
	}

	if err := uc.repo.Create(user); err != nil {
		return nil, err
	}

	// Never return the hash
	user.PasswordHash = nil
	return user, nil
}

func (uc *UserUseCase) Login(email, password string) (string, error) {
	user, err := uc.repo.FindByEmail(email)
	if err != nil {
		return "", err
	}
	if user == nil || user.PasswordHash == nil {
		return "", ErrInvalidCredentials
	}
	if !user.Active {
		return "", ErrUserInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return uc.signToken(user)
}

func (uc *UserUseCase) OAuthLogin(input domain.OAuthUserInput) (string, error) {
	// Check if email belongs to a local account
	existing, err := uc.repo.FindByEmail(input.Email)
	if err != nil {
		return "", err
	}
	if existing != nil && existing.Provider == "local" {
		return "", ErrEmailTakenByLocalAccount
	}
	if existing != nil && !existing.Active {
		return "", ErrUserInactive
	}

	// Check if OAuth user already exists
	user, err := uc.repo.FindByProviderID(input.Provider, input.ProviderID)
	if err != nil {
		return "", err
	}

	if user == nil {
		providerID := input.ProviderID
		user = &domain.User{
			Name:       input.Name,
			Email:      input.Email,
			Role:       "dev",
			Provider:   input.Provider,
			ProviderID: &providerID,
		}
		if err := uc.repo.Create(user); err != nil {
			return "", err
		}
	}

	return uc.signToken(user)
}

func (uc *UserUseCase) signToken(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"role":  user.Role,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(uc.jwtSecret))
}
