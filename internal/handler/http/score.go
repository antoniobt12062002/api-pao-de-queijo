package http

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

type ScoreUseCaseInterface interface {
	GetRanking() ([]*usecase.ScoreResponse, error)
	GetUserScore(userID string) (*usecase.ScoreResponse, error)
	GetUserBadges(userID string) ([]*domain.Badge, error)
}

type ScoreHandler struct {
	uc ScoreUseCaseInterface
}

func NewScoreHandler(uc ScoreUseCaseInterface) *ScoreHandler {
	return &ScoreHandler{uc: uc}
}

// GetRanking godoc
// @Summary      Ranking de score
// @Description  Retorna todos os usuários ordenados por score decrescente
// @Tags         scores
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   usecase.ScoreResponse
// @Failure      401  {object}  ErrInvalidCredentials
// @Router       /scores [get]
func (h *ScoreHandler) GetRanking(w http.ResponseWriter, r *http.Request) {
	ranking, err := h.uc.GetRanking()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, ranking)
}

// GetUserScore godoc
// @Summary      Score de um usuário
// @Description  Retorna o score detalhado de um usuário específico
// @Tags         scores
// @Produce      json
// @Security     BearerAuth
// @Param        user_id  path  string  true  "User ID"
// @Success      200  {object}  usecase.ScoreResponse
// @Failure      401  {object}  ErrInvalidCredentials
// @Failure      404  {object}  ErrValidation
// @Router       /scores/{user_id} [get]
func (h *ScoreHandler) GetUserScore(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	s, err := h.uc.GetUserScore(userID)
	if err != nil {
		if errors.Is(err, usecase.ErrScoreNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// GetUserBadges godoc
// @Summary      Badges de um usuário
// @Description  Retorna todos os badges conquistados por um usuário
// @Tags         scores
// @Produce      json
// @Security     BearerAuth
// @Param        user_id  path  string  true  "User ID"
// @Success      200  {array}   domain.Badge
// @Failure      401  {object}  ErrInvalidCredentials
// @Failure      404  {object}  ErrValidation
// @Router       /badges/{user_id} [get]
func (h *ScoreHandler) GetUserBadges(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	badges, err := h.uc.GetUserBadges(userID)
	if err != nil {
		if errors.Is(err, usecase.ErrScoreNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, badges)
}
