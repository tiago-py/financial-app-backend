package controller

import (
	"net/http"

	"financial-app-backend/internal/repository"
)

func (h *Handler) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.AdminListUsers(r.Context(), UserID(r.Context()))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": users})
}

func (h *Handler) AdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	user, err := h.service.AdminCreateUser(r.Context(), UserID(r.Context()), input.Name, input.Email, input.Password)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (h *Handler) AdminCreateAccount(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name                string `json:"name"`
		Institution         string `json:"institution"`
		Type                string `json:"type"`
		OpeningBalanceCents int64  `json:"openingBalanceCents"`
		OpenedOn            string `json:"openedOn"`
		Color               string `json:"color"`
	}
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	account, err := h.service.AdminCreateAccount(r.Context(), UserID(r.Context()), r.PathValue("userId"), repository.AccountInput{
		Name: input.Name, Institution: input.Institution, Type: input.Type,
		OpeningBalanceCents: input.OpeningBalanceCents, OpenedOn: input.OpenedOn, Color: input.Color,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, account)
}
