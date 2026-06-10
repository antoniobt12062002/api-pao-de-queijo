package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

type AbsenceHandler struct {
	uc *usecase.AbsenceUseCase
}

func NewAbsenceHandler(uc *usecase.AbsenceUseCase) *AbsenceHandler {
	return &AbsenceHandler{uc: uc}
}

// MarkAbsent godoc
// POST /absences  — body: {"date":"YYYY-MM-DD"}
func (h *AbsenceHandler) MarkAbsent(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	var req struct {
		Date string `json:"date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Date == "" {
		writeError(w, http.StatusUnprocessableEntity, "date is required (YYYY-MM-DD)")
		return
	}
	if err := h.uc.MarkAbsent(userID, req.Date); err != nil {
		if errors.Is(err, domain.ErrAbsenceAlreadyExists) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "absence registered"})
}

// RemoveAbsence godoc
// DELETE /absences/{date}
func (h *AbsenceHandler) RemoveAbsence(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	date := chi.URLParam(r, "date")
	if err := h.uc.RemoveAbsence(userID, date); err != nil {
		if errors.Is(err, domain.ErrAbsenceNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "absence removed"})
}

// ListAbsences godoc
// GET /absences — lista ausências futuras do usuário autenticado
func (h *AbsenceHandler) ListAbsences(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	absences, err := h.uc.ListAbsences(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, absences)
}
