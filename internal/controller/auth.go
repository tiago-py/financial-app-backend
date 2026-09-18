package controller

import (
	"net/http"
	"strings"

	"financial-app-backend/internal/model"
)

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input authRequest
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	user, token, err := h.service.Login(r.Context(), input.Email, input.Password)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setSessionCookie(w, r, token)
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "accessToken": token, "tokenType": "Bearer"})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.Me(r.Context(), UserID(r.Context()))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Timezone string `json:"timezone"`
		Currency string `json:"currency"`
	}
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	user, err := h.service.UpdateMe(r.Context(), UserID(r.Context()), input.Name, input.Timezone, input.Currency)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: "financial_session", Value: "", Path: "/", HttpOnly: true,
		Secure: secureRequest(r), SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: "financial_session", Value: token, Path: "/", HttpOnly: true,
		Secure: secureRequest(r), SameSite: http.SameSiteLaxMode, MaxAge: 86400,
	})
}

func secureRequest(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Ping(r.Context()); err != nil {
		writeError(w, r, model.ErrInvalid)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
