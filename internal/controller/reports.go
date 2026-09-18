package controller

import (
	"net/http"
	"time"
)

func reportDates(r *http.Request) (string, string) {
	now := time.Now()
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from == "" {
		from = first.Format("2006-01-02")
	}
	if to == "" {
		to = now.Format("2006-01-02")
	}
	return from, to
}

func (h *Handler) SpendingReport(w http.ResponseWriter, r *http.Request) {
	from, to := reportDates(r)
	result, err := h.service.Spending(r.Context(), UserID(r.Context()), from, to)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) MonthlyAverage(w http.ResponseWriter, r *http.Request) {
	from, to := reportDates(r)
	amount, months, err := h.service.MonthlyAverage(r.Context(), UserID(r.Context()), from, to)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from": from, "to": to, "averageAmountCents": amount, "monthsConsidered": months, "currency": "BRL",
	})
}

func (h *Handler) Projection(w http.ResponseWriter, r *http.Request) {
	var input struct {
		BaseDate string `json:"baseDate"`
		Months   int    `json:"months"`
	}
	if decodeJSON(r, &input) != nil {
		badJSON(w, r)
		return
	}
	result, err := h.service.Projection(r.Context(), UserID(r.Context()), input.BaseDate, input.Months)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"baseDate": input.BaseDate, "months": input.Months, "currency": "BRL",
		"assumptions": []string{"Saldo atual confirmado", "Fluxos planejados ainda nao realizados"},
		"points":      result,
	})
}
