package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

type UserHandler struct {
	uc *usecase.UserUseCase
}

func NewUserHandler(uc *usecase.UserUseCase) *UserHandler {
	return &UserHandler{uc: uc}
}

type registerRequest struct {
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Role     string  `json:"role"`
	Phone    *string `json:"phone"`
}

// Register godoc
// @Summary      Cadastrar usuário
// @Description  Cria um novo usuário com email e senha
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      registerRequest       true  "Dados do usuário"
// @Success      201   {object}  domain.User
// @Failure      409   {object}  ErrEmailConflict  "email already registered"
// @Failure      422   {object}  ErrValidation     "name, email and password are required"
// @Failure      500   {object}  ErrInternal       "internal server error"
// @Router       /users [post]
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusUnprocessableEntity, "name, email and password are required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusUnprocessableEntity, "password must be at least 8 characters")
		return
	}

	user, err := h.uc.CreateUser(domain.CreateUserInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
		Phone:    req.Phone,
	})
	if err != nil {
		if errors.Is(err, usecase.ErrEmailAlreadyExists) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

// ErrorResponse is the standard error envelope returned by all endpoints.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ErrEmailConflict represents a 409 conflict when email is already registered.
type ErrEmailConflict struct {
	Error string `json:"error" example:"email already registered"`
}

// ErrValidation represents a 422 validation error.
type ErrValidation struct {
	Error string `json:"error" example:"name, email and password are required"`
}

// ErrInternal represents a 500 internal server error.
type ErrInternal struct {
	Error string `json:"error" example:"internal server error"`
}

// ErrInvalidCredentials represents a 401 unauthorized error.
type ErrInvalidCredentials struct {
	Error string `json:"error" example:"invalid email or password"`
}

// ErrInvalidBody represents a 422 error for an invalid request body.
type ErrInvalidBody struct {
	Error string `json:"error" example:"invalid request body"`
}

// ErrInvalidState represents a 400 error for an invalid OAuth state parameter.
type ErrInvalidState struct {
	Error string `json:"error" example:"invalid or expired state parameter"`
}

// ErrEmailConflictOAuth represents a 409 conflict when email is already registered via password login.
type ErrEmailConflictOAuth struct {
	Error string `json:"error" example:"email already registered with password login"`
}

// ErrGitHubAuth represents a 502 error when GitHub authentication fails.
type ErrGitHubAuth struct {
	Error string `json:"error" example:"failed to authenticate with GitHub"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
