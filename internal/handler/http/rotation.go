package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

type RotationHandler struct {
	uc *usecase.RotationUseCase
}

func NewRotationHandler(uc *usecase.RotationUseCase) *RotationHandler {
	return &RotationHandler{uc: uc}
}

type updateOrderRequest struct {
	UserIDs []string `json:"user_ids"`
}

// GetCurrent godoc
// @Summary      Listar rodízio atual
// @Description  Retorna a fila circular de pagadores com a posição atual
// @Tags         rotation
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  usecase.RotationResponse
// @Failure      401  {object}  ErrInvalidCredentials
// @Router       /rotation [get]
func (h *RotationHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	resp, err := h.uc.GetCurrent()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// UpdateOrder godoc
// @Summary      Atualizar ordem do rodízio
// @Description  Substitui a fila de pagadores (somente admin)
// @Tags         rotation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      updateOrderRequest  true  "Array completo de user_ids na nova ordem"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  ErrValidation
// @Failure      401   {object}  ErrInvalidCredentials
// @Failure      403   {object}  ErrValidation  "admin role required"
// @Router       /rotation/order [put]
func (h *RotationHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	var req updateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	if err := h.uc.UpdateOrder(req.UserIDs); err != nil {
		switch {
		case errors.Is(err, domain.ErrRotationEmptyOrder),
			errors.Is(err, domain.ErrRotationDuplicateUser),
			errors.Is(err, domain.ErrRotationUnknownUser):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "rotation order updated"})
}

// Skip godoc
// @Summary      Pular vez no rodízio
// @Description  Avança a posição atual da fila para o próximo usuário (somente admin)
// @Tags         rotation
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  ErrInvalidCredentials
// @Failure      403  {object}  ErrValidation  "admin role required"
// @Failure      409  {object}  ErrValidation  "rotation not initialized"
// @Router       /rotation/skip [post]
func (h *RotationHandler) Skip(w http.ResponseWriter, r *http.Request) {
	if err := h.uc.Skip(); err != nil {
		switch {
		case errors.Is(err, domain.ErrRotationNotInitialized):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "rotation advanced"})
}
