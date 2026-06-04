package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
)

type DeviceTokenUseCaseInterface interface {
	RegisterDevice(userID, token, platform string) error
	RemoveDevice(token, callerID string) error
}

type DeviceTokenHandler struct {
	uc DeviceTokenUseCaseInterface
}

func NewDeviceTokenHandler(uc DeviceTokenUseCaseInterface) *DeviceTokenHandler {
	return &DeviceTokenHandler{uc: uc}
}

type registerDeviceRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

// Register godoc
// @Summary      Registrar token FCM
// @Description  Registra ou atualiza um token FCM para o dispositivo do usuário autenticado
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      registerDeviceRequest  true  "Token e plataforma"
// @Success      201   {object}  map[string]string
// @Failure      400   {object}  ErrValidation
// @Failure      401   {object}  ErrInvalidCredentials
// @Router       /devices [post]
func (h *DeviceTokenHandler) Register(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req registerDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" || req.Platform == "" {
		writeError(w, http.StatusBadRequest, "token and platform are required")
		return
	}

	if err := h.uc.RegisterDevice(userID, req.Token, req.Platform); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "device registered"})
}

// Remove godoc
// @Summary      Remover token FCM
// @Description  Remove um token FCM. Apenas o dono do token pode removê-lo.
// @Tags         devices
// @Produce      json
// @Security     BearerAuth
// @Param        token  path  string  true  "FCM token"
// @Success      204
// @Failure      401    {object}  ErrInvalidCredentials
// @Failure      403    {object}  ErrValidation
// @Failure      404    {object}  ErrValidation
// @Router       /devices/{token} [delete]
func (h *DeviceTokenHandler) Remove(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	callerID := UserIDFromContext(r.Context())

	if err := h.uc.RemoveDevice(token, callerID); err != nil {
		switch {
		case errors.Is(err, domain.ErrDeviceTokenNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrDeviceTokenForbidden):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
