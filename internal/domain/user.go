package domain

import "time"

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash *string   `json:"-"`
	Role         string    `json:"role"`
	Phone        *string   `json:"phone"`
	Provider     string    `json:"provider"`
	ProviderID   *string   `json:"-"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateUserInput struct {
	Name     string
	Email    string
	Password string
	Role     string
	Phone    *string
}

type OAuthUserInput struct {
	Name       string
	Email      string
	Provider   string
	ProviderID string
}

type UserRepository interface {
	Create(user *User) error
	FindByEmail(email string) (*User, error)
	FindByProviderID(provider, providerID string) (*User, error)
	FindByID(id string) (*User, error)
	FindAll() ([]*User, error)
	UpdateRole(id, role string) error
	Deactivate(id string) error
}
