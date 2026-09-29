package controller

import (
	"net/http"
	"strconv"

	"financial-app-backend/internal/repository"
)

type accountRequest struct {
	Name                string `json:"name"`
	Institution         string `json:"institution"`
	Type                string `json:"type"`
	OpeningBalanceCents int64  `json:"openingBalanceCents"`
	OpenedOn            string `json:"openedOn"`
	Color               string `json:"color"`
}

func accountInput(input accountRequest) repository.AccountInput {
	return repository.AccountInput{
		Name: input.Name, Institution: input.Institution, Type: input.Type,
		OpeningBalanceCents: input.OpeningBalanceCents, OpenedOn: input.OpenedOn, Color: input.Color,
	}
}

func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListAccounts(r.Context(), UserID(r.Context()), r.URL.Query().Get("archived") == "true")
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetAccount(r.Context(), UserID(r.Context()), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	var input accountRequest
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.UpdateAccount(r.Context(), UserID(r.Context()), r.PathValue("id"), accountInput(input))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) ArchiveAccount(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.ArchiveAccount(r.Context(), UserID(r.Context()), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) RestoreAccount(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.RestoreAccount(r.Context(), UserID(r.Context()), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) Balances(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Balances(r.Context(), UserID(r.Context()))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type categoryRequest struct {
	Name    string `json:"name"`
	Purpose string `json:"purpose"`
	Color   string `json:"color"`
}

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListCategories(r.Context(), UserID(r.Context()), r.URL.Query().Get("purpose"), r.URL.Query().Get("archived") == "true")
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var input categoryRequest
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.CreateCategory(r.Context(), UserID(r.Context()), repository.CategoryInput{
		Name: input.Name, Purpose: input.Purpose, Color: input.Color,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	var input categoryRequest
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.UpdateCategory(r.Context(), UserID(r.Context()), r.PathValue("id"), repository.CategoryInput{
		Name: input.Name, Purpose: input.Purpose, Color: input.Color,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) ArchiveCategory(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.ArchiveCategory(r.Context(), UserID(r.Context()), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func queryInt(r *http.Request, key string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return fallback
	}
	return value
}

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var input accountRequest
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	account, err := h.service.CreateAccount(r.Context(), UserID(r.Context()), accountInput(input))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, account)
}
