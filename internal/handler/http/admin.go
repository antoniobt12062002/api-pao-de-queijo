package http

import (
	"net/http"

	"github.com/antoniobt12062002/pao-de-queijo/internal/job"
)

type AdminHandler struct {
	roundCreator *job.DailyRoundCreator
}

func NewAdminHandler(roundCreator *job.DailyRoundCreator) *AdminHandler {
	return &AdminHandler{roundCreator: roundCreator}
}

func (h *AdminHandler) TriggerRound(w http.ResponseWriter, r *http.Request) {
	h.roundCreator.Run()
	writeJSON(w, http.StatusOK, map[string]string{"message": "round creation triggered"})
}
