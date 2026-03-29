package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

type ConfigHandler struct {
	uc *usecase.ConfigUseCase
}

func NewConfigHandler(uc *usecase.ConfigUseCase) *ConfigHandler {
	return &ConfigHandler{uc: uc}
}

type updateConfigRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// GetAll godoc
// @Summary      Listar configurações
// @Description  Retorna todas as configurações globais do sistema
// @Tags         config
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   domain.Config
// @Failure      401  {object}  ErrInvalidCredentials
// @Router       /config [get]
func (h *ConfigHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	configs, err := h.uc.GetAll()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, configs)
}

// Update godoc
// @Summary      Atualizar configuração
// @Description  Atualiza o valor de uma chave de configuração (somente admin)
// @Tags         config
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      updateConfigRequest  true  "Chave e valor"
// @Success      200   {object}  domain.Config
// @Failure      400   {object}  ErrValidation
// @Failure      401   {object}  ErrInvalidCredentials
// @Failure      403   {object}  ErrValidation  "admin role required"
// @Failure      422   {object}  ErrValidation
// @Router       /config [put]
func (h *ConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if req.Key == "" || req.Value == "" {
		writeError(w, http.StatusUnprocessableEntity, "key and value are required")
		return
	}

	if err := h.uc.Update(req.Key, req.Value); err != nil {
		switch {
		case errors.Is(err, domain.ErrConfigUnknownKey):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrConfigInvalidValue):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, domain.Config{Key: req.Key, Value: req.Value})
}
