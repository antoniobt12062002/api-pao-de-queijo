package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

type ParticipationHandler struct {
	uc *usecase.ParticipationUseCase
}

func NewParticipationHandler(uc *usecase.ParticipationUseCase) *ParticipationHandler {
	return &ParticipationHandler{uc: uc}
}

type participateRequest struct {
	Quantity int `json:"quantity"`
}

// Participate godoc
// @Summary      Participar da rodada
// @Description  Usuário registra participação na rodada aberta com a quantidade desejada
// @Tags         participations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string              true  "Round ID"
// @Param        body  body      participateRequest  true  "Quantidade"
// @Success      201   {object}  map[string]string
// @Failure      400   {object}  ErrValidation
// @Failure      401   {object}  ErrInvalidCredentials
// @Failure      404   {object}  ErrValidation
// @Failure      409   {object}  ErrValidation  "round not open or already participating"
// @Router       /rounds/{id}/participate [post]
func (h *ParticipationHandler) Participate(w http.ResponseWriter, r *http.Request) {
	roundID := chi.URLParam(r, "id")
	userID := UserIDFromContext(r.Context())

	var req participateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Quantity < 1 {
		writeError(w, http.StatusBadRequest, "quantity is required and must be >= 1")
		return
	}

	if err := h.uc.Participate(roundID, userID, req.Quantity); err != nil {
		switch {
		case errors.Is(err, domain.ErrRoundNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrRoundNotOpen):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrAlreadyParticipating):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "participation registered"})
}

// Withdraw godoc
// @Summary      Retirar participação
// @Description  Usuário cancela sua participação na rodada aberta
// @Tags         participations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Round ID"
// @Success      204
// @Failure      401  {object}  ErrInvalidCredentials
// @Failure      404  {object}  ErrValidation
// @Failure      409  {object}  ErrValidation  "round not open"
// @Router       /rounds/{id}/participate [delete]
func (h *ParticipationHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	roundID := chi.URLParam(r, "id")
	userID := UserIDFromContext(r.Context())

	if err := h.uc.Withdraw(roundID, userID); err != nil {
		switch {
		case errors.Is(err, domain.ErrRoundNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrParticipationNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrRoundNotOpen):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetParticipations godoc
// @Summary      Listar participações
// @Description  Lista todos os participantes da rodada e o total de pães
// @Tags         participations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Round ID"
// @Success      200  {object}  usecase.ParticipationsResponse
// @Failure      401  {object}  ErrInvalidCredentials
// @Router       /rounds/{id}/participations [get]
func (h *ParticipationHandler) GetParticipations(w http.ResponseWriter, r *http.Request) {
	roundID := chi.URLParam(r, "id")

	resp, err := h.uc.GetParticipations(roundID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
