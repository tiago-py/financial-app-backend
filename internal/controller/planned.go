package controller

import (
	"net/http"

	"financial-app-backend/internal/repository"
)

type plannedRequest struct {
	AccountID   *string `json:"accountId"`
	CategoryID  *string `json:"categoryId"`
	Direction   string  `json:"direction"`
	AmountCents int64   `json:"amountCents"`
	ExpectedOn  string  `json:"expectedOn"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
}

func plannedInput(input plannedRequest) repository.PlannedInput {
	return repository.PlannedInput{
		AccountID: input.AccountID, CategoryID: input.CategoryID, Direction: input.Direction,
		AmountCents: input.AmountCents, ExpectedOn: input.ExpectedOn,
		Description: input.Description, Status: input.Status,
	}
}

func (h *Handler) ListPlanned(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	items, err := h.service.ListPlanned(r.Context(), UserID(r.Context()), query.Get("status"), query.Get("from"), query.Get("to"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreatePlanned(w http.ResponseWriter, r *http.Request) {
	var input plannedRequest
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.CreatePlanned(r.Context(), UserID(r.Context()), plannedInput(input))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdatePlanned(w http.ResponseWriter, r *http.Request) {
	var input plannedRequest
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.UpdatePlanned(r.Context(), UserID(r.Context()), r.PathValue("id"), plannedInput(input))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) DeletePlanned(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeletePlanned(r.Context(), UserID(r.Context()), r.PathValue("id")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RealizePlanned(w http.ResponseWriter, r *http.Request) {
	var input struct {
		AccountID  string `json:"accountId"`
		OccurredOn string `json:"occurredOn"`
	}
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.RealizePlanned(r.Context(), UserID(r.Context()), r.PathValue("id"), input.AccountID, input.OccurredOn)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
