package controller

import (
	"net/http"

	"financial-app-backend/internal/repository"
)

type debtRequest struct {
	Description      string  `json:"description"`
	Creditor         string  `json:"creditor"`
	PrincipalCents   int64   `json:"principalCents"`
	DueDate          *string `json:"dueDate"`
	Status           string  `json:"status"`
	InstallmentCount int     `json:"installmentCount"`
}

func debtInput(input debtRequest) repository.DebtInput {
	return repository.DebtInput{
		Description: input.Description, Creditor: input.Creditor,
		PrincipalCents: input.PrincipalCents, DueDate: input.DueDate, Status: input.Status,
	}
}

func (h *Handler) ListDebts(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListDebts(r.Context(), UserID(r.Context()), r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) GetDebt(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetDebt(r.Context(), UserID(r.Context()), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) CreateDebt(w http.ResponseWriter, r *http.Request) {
	var input debtRequest
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.CreateDebt(r.Context(), UserID(r.Context()), debtInput(input), input.InstallmentCount)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) ListDebtInstallments(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListDebtInstallments(r.Context(), UserID(r.Context()), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) UpdateDebt(w http.ResponseWriter, r *http.Request) {
	var input debtRequest
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.UpdateDebt(r.Context(), UserID(r.Context()), r.PathValue("id"), debtInput(input))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteDebt(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeleteDebt(r.Context(), UserID(r.Context()), r.PathValue("id")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListPayments(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListPayments(r.Context(), UserID(r.Context()), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var input struct {
		AccountID   string `json:"accountId"`
		AmountCents int64  `json:"amountCents"`
		PaidOn      string `json:"paidOn"`
	}
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.CreatePayment(r.Context(), UserID(r.Context()), r.PathValue("id"),
		input.AccountID, input.AmountCents, input.PaidOn, r.Header.Get("Idempotency-Key"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) ReversePayment(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Reason     string `json:"reason"`
		OccurredOn string `json:"occurredOn"`
	}
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	item, err := h.service.ReversePayment(r.Context(), UserID(r.Context()), r.PathValue("id"), input.Reason, input.OccurredOn)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
