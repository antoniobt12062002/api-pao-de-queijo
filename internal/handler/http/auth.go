package http

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
)

type stateEntry struct {
	createdAt time.Time
}

type AuthHandler struct {
	uc                *usecase.UserUseCase
	githubClientID    string
	githubSecret      string
	githubCallbackURL string
	states            map[string]stateEntry
	mu                sync.Mutex
}

func NewAuthHandler(uc *usecase.UserUseCase, clientID, secret, callbackURL string) *AuthHandler {
	return &AuthHandler{
		uc:                uc,
		githubClientID:    clientID,
		githubSecret:      secret,
		githubCallbackURL: callbackURL,
		states:            make(map[string]stateEntry),
	}
}

// loginRequest is used by swag for documentation
type loginRequest struct {
	Email    string `json:"email"    example:"joao@empresa.com"`
	Password string `json:"password" example:"s3nh4segura"`
}

// tokenResponse is the JWT token envelope returned on successful auth.
type tokenResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// Login godoc
// @Summary      Login com email e senha
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest          true  "Credenciais"
// @Success      200   {object}  tokenResponse     "token JWT"
// @Failure      401   {object}  ErrInvalidCredentials  "invalid email or password"
// @Failure      422   {object}  ErrInvalidBody         "invalid request body"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	token, err := h.uc.Login(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{Token: token})
}

// GitHubLogin godoc
// @Summary      Iniciar login com GitHub
// @Description  Redireciona para o GitHub OAuth
// @Tags         auth
// @Success      302  "Redireciona para GitHub"
// @Router       /auth/github [get]
func (h *AuthHandler) GitHubLogin(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	h.mu.Lock()
	h.states[state] = stateEntry{createdAt: time.Now()}
	h.mu.Unlock()

	redirectURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=user:email&state=%s",
		h.githubClientID,
		url.QueryEscape(h.githubCallbackURL),
		state,
	)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// GitHubCallback godoc
// @Summary      Callback do GitHub OAuth
// @Description  GitHub chama este endpoint após autorização
// @Tags         auth
// @Produce      json
// @Param        code   query  string  true  "Código do GitHub"
// @Param        state  query  string  true  "State anti-CSRF"
// @Success      200    {object}  tokenResponse  "token JWT"
// @Failure      400    {object}  ErrInvalidState       "invalid or expired state parameter"
// @Failure      409    {object}  ErrEmailConflictOAuth "email already registered with password login"
// @Failure      500    {object}  ErrInternal           "internal server error"
// @Failure      502    {object}  ErrGitHubAuth         "failed to authenticate with GitHub"
// @Router       /auth/github/callback [get]
func (h *AuthHandler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	h.mu.Lock()
	entry, ok := h.states[state]
	if ok {
		delete(h.states, state)
	}
	h.mu.Unlock()

	if !ok || time.Since(entry.createdAt) > 10*time.Minute {
		writeError(w, http.StatusBadRequest, "invalid or expired state parameter")
		return
	}

	accessToken, err := h.exchangeGitHubCode(code)
	if err != nil {
		slog.Error("github token exchange failed", "err", err)
		writeError(w, http.StatusBadGateway, "failed to authenticate with GitHub")
		return
	}

	ghUser, err := h.fetchGitHubUser(accessToken)
	if err != nil {
		slog.Error("github user fetch failed", "err", err)
		writeError(w, http.StatusBadGateway, "failed to fetch GitHub user")
		return
	}

	token, err := h.uc.OAuthLogin(domain.OAuthUserInput{
		Name:       ghUser.Name,
		Email:      ghUser.Email,
		Provider:   "github",
		ProviderID: fmt.Sprintf("%d", ghUser.ID),
	})
	if err != nil {
		if errors.Is(err, usecase.ErrEmailTakenByLocalAccount) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{Token: token})
}

type githubUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (h *AuthHandler) exchangeGitHubCode(code string) (string, error) {
	body := fmt.Sprintf("client_id=%s&client_secret=%s&code=%s",
		h.githubClientID, h.githubSecret, code)

	req, _ := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token",
		strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("github error: %s", result.Error)
	}
	return result.AccessToken, nil
}

func (h *AuthHandler) fetchGitHubUser(accessToken string) (*githubUser, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var u githubUser
	if err := json.Unmarshal(data, &u); err != nil {
		return nil, err
	}

	// If email is not public, fetch from /user/emails
	if u.Email == "" {
		u.Email, err = h.fetchPrimaryEmail(accessToken)
		if err != nil {
			return nil, err
		}
	}
	return &u, nil
}

func (h *AuthHandler) fetchPrimaryEmail(accessToken string) (string, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}
	for _, e := range emails {
		if e.Primary {
			return e.Email, nil
		}
	}
	return "", fmt.Errorf("no primary email found on GitHub account")
}

func generateState() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(b)
}
