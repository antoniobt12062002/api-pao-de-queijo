package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/job"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

type AdminHandler struct {
	creator   *job.DailyRoundCreator
	closer    *job.ParticipationWindowCloser
	roundUC   *usecase.RoundUseCase
	userRepo  domain.UserRepository
	notifySvc domain.NotificationService
}

func NewAdminHandler(
	creator *job.DailyRoundCreator,
	closer *job.ParticipationWindowCloser,
	roundUC *usecase.RoundUseCase,
	userRepo domain.UserRepository,
	notifySvc domain.NotificationService,
) *AdminHandler {
	return &AdminHandler{creator: creator, closer: closer, roundUC: roundUC, userRepo: userRepo, notifySvc: notifySvc}
}

func (h *AdminHandler) TriggerRound(w http.ResponseWriter, r *http.Request) {
	h.creator.Run()
	writeJSON(w, http.StatusOK, map[string]string{"message": "round creation triggered"})
}

func (h *AdminHandler) CreateRound(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Date     string `json:"date"`
		PayerID  string `json:"payer_id"`
		NotifyAt string `json:"notify_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Date == "" {
		writeError(w, http.StatusUnprocessableEntity, "date is required (YYYY-MM-DD)")
		return
	}
	if err := h.creator.CreateForDate(req.Date, req.PayerID, req.NotifyAt); err != nil {
		switch {
		case errors.Is(err, domain.ErrRoundAlreadyExists):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrRotationEmpty):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "round created"})
}

func (h *AdminHandler) ForceCloseRound(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.closer.CloseRoundByID(id); err != nil {
		switch {
		case errors.Is(err, domain.ErrRoundNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrRoundNotOpen):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "round closed"})
}

func (h *AdminHandler) ChangePayer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		writeError(w, http.StatusUnprocessableEntity, "user_id is required")
		return
	}
	if err := h.roundUC.ChangePayer(id, req.UserID); err != nil {
		switch {
		case errors.Is(err, domain.ErrRoundNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrRoundNotPending):
			writeError(w, http.StatusConflict, "round cannot be modified in its current status")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "payer updated"})
}

func (h *AdminHandler) SendNotification(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserIDs []string `json:"user_ids"`
		Title   string   `json:"title"`
		Body    string   `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if req.Title == "" || req.Body == "" {
		writeError(w, http.StatusUnprocessableEntity, "title and body are required")
		return
	}

	userIDs := req.UserIDs
	if len(userIDs) == 0 {
		users, err := h.userRepo.FindAll()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		for _, u := range users {
			if u.Active {
				userIDs = append(userIDs, u.ID)
			}
		}
	}

	if err := h.notifySvc.SendManual(userIDs, req.Title, req.Body); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send notification")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "notification sent", "recipients": len(userIDs)})
}
