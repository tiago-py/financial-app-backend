package controller

import (
	"net/http"

	"financial-app-backend/internal/repository"
)

type entryRequest struct {
	AccountID   string  `json:"accountId"`
	CategoryID  *string `json:"categoryId"`
	AmountCents int64   `json:"amountCents"`
	OccurredOn  string  `json:"occurredOn"`
	Description string  `json:"description"`
}

func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	items, total, err := h.service.ListTransactions(r.Context(), UserID(r.Context()), repository.TransactionFilter{
		From: query.Get("from"), To: query.Get("to"), AccountID: query.Get("accountId"),
		CategoryID: query.Get("categoryId"), Kind: query.Get("type"),
		Limit: queryInt(r, "limit", 30), Offset: queryInt(r, "offset", 0),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (h *Handler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetTransaction(r.Context(), UserID(r.Context()), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) CreateIncome(w http.ResponseWriter, r *http.Request) {
	h.createEntry(w, r, "income")
}

func (h *Handler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	h.createEntry(w, r, "expense")
}

func (h *Handler) createEntry(w http.ResponseWriter, r *http.Request, kind string) {
	var input entryRequest
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.CreateTransaction(r.Context(), UserID(r.Context()), repository.EntryInput{
		AccountID: input.AccountID, CategoryID: input.CategoryID, AmountCents: input.AmountCents,
		Kind: kind, OccurredOn: input.OccurredOn, Description: input.Description,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Description string  `json:"description"`
		CategoryID  *string `json:"categoryId"`
	}
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.UpdateTransaction(r.Context(), UserID(r.Context()), r.PathValue("id"), input.Description, input.CategoryID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) ReverseTransaction(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Reason     string `json:"reason"`
		OccurredOn string `json:"occurredOn"`
	}
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.ReverseTransaction(r.Context(), UserID(r.Context()), r.PathValue("id"), input.Reason, input.OccurredOn)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

type transferRequest struct {
	FromAccountID string `json:"fromAccountId"`
	ToAccountID   string `json:"toAccountId"`
	AmountCents   int64  `json:"amountCents"`
	OccurredOn    string `json:"occurredOn"`
	Description   string `json:"description"`
}

func (h *Handler) ListTransfers(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListTransfers(r.Context(), UserID(r.Context()))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	var input transferRequest
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.CreateTransfer(r.Context(), UserID(r.Context()), repository.TransferInput{
		FromAccountID: input.FromAccountID, ToAccountID: input.ToAccountID,
		AmountCents: input.AmountCents, OccurredOn: input.OccurredOn, Description: input.Description,
	}, r.Header.Get("Idempotency-Key"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) ReverseTransfer(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Reason     string `json:"reason"`
		OccurredOn string `json:"occurredOn"`
	}
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.ReverseTransfer(r.Context(), UserID(r.Context()), r.PathValue("id"), input.Reason, input.OccurredOn)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
