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
// @Failure      409   {object}  ErrorResponse  "email already registered"
// @Failure      422   {object}  ErrorResponse  "name, email and password are required"
// @Failure      500   {object}  ErrorResponse  "internal server error"
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
	Error string `json:"error" example:"email already registered"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
