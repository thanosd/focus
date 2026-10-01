package handlers

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/thanosd/focus/backend/internal/api"
	"github.com/thanosd/focus/backend/internal/config"
	"github.com/thanosd/focus/backend/internal/services"
)

// AuthHandler serves sign-in, session and API token endpoints.
type AuthHandler struct {
	cfg  *config.Config
	auth *services.AuthService
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(cfg *config.Config, auth *services.AuthService) *AuthHandler {
	return &AuthHandler{cfg: cfg, auth: auth}
}

// RequireSession resolves the session cookie and stores the user on the
// context, or answers 401.
func (h *AuthHandler) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := h.auth.UserFromRequest(r)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
	})
}

// HandleGoogleLogin starts the OAuth round trip.
func (h *AuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !h.auth.SignInConfigured() {
		writeJSONError(w, http.StatusServiceUnavailable, "Google sign-in is not configured")
		return
	}
	authURL, err := h.auth.BeginGoogleLogin(r.Context(), r.URL.Query().Get("redirect"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

// HandleGoogleCallback finishes the OAuth round trip and redirects to the app.
func (h *AuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		log.Printf("auth: google returned error %q", e)
		h.redirectLoginError(w, r, services.AuthErrOAuthFailed)
		return
	}
	sessionID, redirectTo, err := h.auth.CompleteGoogleLogin(r.Context(), q.Get("state"), q.Get("code"))
	if err != nil {
		code := services.AuthErrOAuthFailed
		switch {
		case errors.Is(err, services.ErrEmailNotAllowed):
			code = services.AuthErrNotAllowed
		case errors.Is(err, services.ErrInvalidState):
			code = services.AuthErrStateMismatch
		default:
			log.Printf("auth: google callback failed: %v", err)
		}
		h.redirectLoginError(w, r, code)
		return
	}
	h.setSessionCookie(w, sessionID, int(h.auth.SessionDuration().Seconds()))
	http.Redirect(w, r, h.cfg.FrontendURL+redirectTo, http.StatusFound)
}

func (h *AuthHandler) redirectLoginError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, h.cfg.FrontendURL+"/login?error="+url.QueryEscape(code), http.StatusFound)
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     services.SessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.SecureCookies(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
}

// HandleLogout deletes the session and clears the cookie.
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(services.SessionCookieName); err == nil {
		if err := h.auth.Logout(r.Context(), cookie.Value); err != nil {
			writeServiceError(w, err)
			return
		}
	}
	h.setSessionCookie(w, "", -1)
	writeJSON(w, http.StatusOK, api.OkResponse{Ok: true})
}

// HandleMe returns the current user.
func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	user, err := h.auth.UserFromRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, api.MeResponse{User: toAPIUser(user)})
}

// HandleGetPreferences returns user preferences.
func (h *AuthHandler) HandleGetPreferences(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	writeJSON(w, http.StatusOK, api.UserPreferences{Timezone: user.Timezone})
}

// HandleUpdatePreferences updates user preferences.
func (h *AuthHandler) HandleUpdatePreferences(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	var req api.UpdateUserPreferencesRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Timezone != nil {
		if err := h.auth.UpdateTimezone(r.Context(), user.ID, *req.Timezone); err != nil {
			writeServiceError(w, err)
			return
		}
		user.Timezone = strings.TrimSpace(*req.Timezone)
	}
	writeJSON(w, http.StatusOK, api.UserPreferences{Timezone: user.Timezone})
}

// HandleListAPITokens lists the user's tokens.
func (h *AuthHandler) HandleListAPITokens(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	tokens, err := h.auth.ListAPITokens(r.Context(), user.ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]api.ApiToken, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, toAPIToken(t))
	}
	writeJSON(w, http.StatusOK, out)
}

// HandleCreateAPIToken mints a token and returns the secret once.
func (h *AuthHandler) HandleCreateAPIToken(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	var req api.CreateApiTokenRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	plaintext, token, err := h.auth.CreateAPIToken(r.Context(), user.ID, req.Name)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, api.CreateApiTokenResponse{Token: plaintext, ApiToken: toAPIToken(*token)})
}

// HandleDeleteAPIToken revokes a token.
func (h *AuthHandler) HandleDeleteAPIToken(w http.ResponseWriter, r *http.Request) {
	user := mustUser(w, r)
	if user == nil {
		return
	}
	id, ok := pathUUID(w, r, "tokenId")
	if !ok {
		return
	}
	if err := h.auth.RevokeAPIToken(r.Context(), user.ID, id); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, api.OkResponse{Ok: true})
}
