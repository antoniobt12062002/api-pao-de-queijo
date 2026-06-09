package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

type RoundHandler struct {
	uc *usecase.RoundUseCase
}

func NewRoundHandler(uc *usecase.RoundUseCase) *RoundHandler {
	return &RoundHandler{uc: uc}
}

// GetAll godoc
// @Summary      Listar rodadas
// @Description  Retorna histórico paginado de rodadas
// @Tags         rounds
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int  false  "Página (default 1)"
// @Param        limit  query     int  false  "Itens por página (default 20, max 100)"
// @Success      200    {object}  usecase.PaginatedRoundsResponse
// @Failure      401    {object}  ErrInvalidCredentials
// @Router       /rounds [get]
func (h *RoundHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	page := parseIntParam(r.URL.Query().Get("page"), 1)
	limit := parseIntParam(r.URL.Query().Get("limit"), 20)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	resp, err := h.uc.GetAll(page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetToday godoc
// @Summary      Rodada de hoje
// @Description  Retorna a rodada do dia atual com campo is_payer para o usuário autenticado
// @Tags         rounds
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  usecase.TodayRoundResponse
// @Failure      401  {object}  ErrInvalidCredentials
// @Router       /rounds/today [get]
func (h *RoundHandler) GetToday(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	date := r.URL.Query().Get("date") // optional: client's local date (YYYY-MM-DD)
	resp, err := h.uc.GetToday(userID, date)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Confirm godoc
// @Summary      Confirmar pagamento
// @Description  Pagador confirma a rodada, abrindo para participações
// @Tags         rounds
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Round ID"
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  ErrInvalidCredentials
// @Failure      403  {object}  ErrValidation  "payer only"
// @Failure      404  {object}  ErrValidation
// @Failure      409  {object}  ErrValidation  "round not in pending status"
// @Router       /rounds/{id}/confirm [post]
func (h *RoundHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := UserIDFromContext(r.Context())

	if err := h.uc.Confirm(id, userID); err != nil {
		switch {
		case errors.Is(err, domain.ErrRoundNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrRoundNotPayer):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, domain.ErrRoundNotPending):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "round confirmed"})
}

// Cancel godoc
// @Summary      Cancelar pagamento
// @Description  Pagador cancela a rodada; próximo da fila é notificado
// @Tags         rounds
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Round ID"
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  ErrInvalidCredentials
// @Failure      403  {object}  ErrValidation  "payer only"
// @Failure      404  {object}  ErrValidation
// @Failure      409  {object}  ErrValidation  "round not in pending status"
// @Router       /rounds/{id}/cancel [post]
func (h *RoundHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := UserIDFromContext(r.Context())

	if err := h.uc.Cancel(id, userID); err != nil {
		switch {
		case errors.Is(err, domain.ErrRoundNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrRoundNotPayer):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, domain.ErrRoundNotPending):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "round cancelled and reassigned"})
}

func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return n
}
