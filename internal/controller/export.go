package controller

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"

	"financial-app-backend/internal/repository"
)

func (h *Handler) ExportTransactions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	items, err := h.service.ExportTransactions(r.Context(), UserID(r.Context()), repository.TransactionFilter{
		From: query.Get("from"), To: query.Get("to"), AccountID: query.Get("accountId"),
		CategoryID: query.Get("categoryId"), Kind: query.Get("type"),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="transacoes.csv"`)
	w.Header().Set("Cache-Control", "no-store")
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	_ = writer.Write([]string{"id", "data", "descricao", "conta", "categoria", "tipo", "direcao", "valor_centavos", "moeda"})
	for _, item := range items {
		category := ""
		if item.CategoryName != nil {
			category = *item.CategoryName
		}
		_ = writer.Write([]string{
			item.ID,
			item.OccurredOn,
			safeCSV(item.Description),
			safeCSV(item.AccountName),
			safeCSV(category),
			item.Kind,
			item.Direction,
			strconv.FormatInt(item.AmountCents, 10),
			item.Currency,
		})
	}
	writer.Flush()
}

func safeCSV(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}
