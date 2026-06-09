package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	handler "github.com/antoniobt12062002/pao-de-queijo/internal/handler/http"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

// --- stub use case ---

type stubScoreUC struct {
	ranking    []*usecase.ScoreResponse
	userScore  *usecase.ScoreResponse
	badges     []*domain.Badge
	rankingErr error
	scoreErr   error
	badgesErr  error
}

func (s *stubScoreUC) GetRanking() ([]*usecase.ScoreResponse, error) {
	return s.ranking, s.rankingErr
}
func (s *stubScoreUC) GetUserScore(userID string) (*usecase.ScoreResponse, error) {
	return s.userScore, s.scoreErr
}
func (s *stubScoreUC) GetUserBadges(userID string) ([]*domain.Badge, error) {
	return s.badges, s.badgesErr
}
func (s *stubScoreUC) GetJusticeChart() ([]*usecase.JusticeEntry, error) { return nil, nil }

// --- tests ---

func TestScoreHandler_GetRanking_OK(t *testing.T) {
	stub := &stubScoreUC{
		ranking: []*usecase.ScoreResponse{
			{Score: &domain.Score{UserID: "u1", Score: 75}, UserName: "Alice"},
		},
	}
	h := handler.NewScoreHandler(stub)

	req := httptest.NewRequest(http.MethodGet, "/v1/scores", nil)
	req = withUserID(req, "u1")
	w := httptest.NewRecorder()
	h.GetRanking(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestScoreHandler_GetUserScore_OK(t *testing.T) {
	stub := &stubScoreUC{
		userScore: &usecase.ScoreResponse{
			Score: &domain.Score{UserID: "u1", Score: 60}, UserName: "Alice",
		},
	}
	h := handler.NewScoreHandler(stub)

	r := chi.NewRouter()
	r.Get("/v1/scores/{user_id}", h.GetUserScore)

	req := httptest.NewRequest(http.MethodGet, "/v1/scores/u1", nil)
	req = withUserID(req, "u1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestScoreHandler_GetUserScore_NotFound(t *testing.T) {
	stub := &stubScoreUC{scoreErr: usecase.ErrScoreNotFound}
	h := handler.NewScoreHandler(stub)

	r := chi.NewRouter()
	r.Get("/v1/scores/{user_id}", h.GetUserScore)

	req := httptest.NewRequest(http.MethodGet, "/v1/scores/nonexistent", nil)
	req = withUserID(req, "u1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestScoreHandler_GetUserBadges_OK(t *testing.T) {
	stub := &stubScoreUC{
		badges: []*domain.Badge{
			{ID: "b1", UserID: "u1", Type: domain.BadgeNovoNaFila, EarnedAt: time.Now()},
		},
	}
	h := handler.NewScoreHandler(stub)

	r := chi.NewRouter()
	r.Get("/v1/badges/{user_id}", h.GetUserBadges)

	req := httptest.NewRequest(http.MethodGet, "/v1/badges/u1", nil)
	req = withUserID(req, "u1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var badges []*domain.Badge
	if err := json.NewDecoder(w.Body).Decode(&badges); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(badges) != 1 {
		t.Errorf("expected 1 badge, got %d", len(badges))
	}
}

func TestScoreHandler_GetUserBadges_NotFound(t *testing.T) {
	stub := &stubScoreUC{badgesErr: usecase.ErrScoreNotFound}
	h := handler.NewScoreHandler(stub)

	r := chi.NewRouter()
	r.Get("/v1/badges/{user_id}", h.GetUserBadges)

	req := httptest.NewRequest(http.MethodGet, "/v1/badges/nonexistent", nil)
	req = withUserID(req, "u1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
