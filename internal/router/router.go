package router

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"financial-app-backend/internal/controller"
	"financial-app-backend/internal/service"
)

func New(handler *controller.Handler, service *service.Service, corsOrigin string, logger *slog.Logger) http.Handler {
	public := http.NewServeMux()
	public.HandleFunc("GET /health", handler.Health)
	public.HandleFunc("POST /api/v1/auth/login", handler.Login)

	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/v1/me", handler.Me)
	protected.HandleFunc("PATCH /api/v1/me", handler.UpdateMe)
	protected.HandleFunc("PATCH /api/v1/me/password", handler.ChangePassword)
	protected.HandleFunc("POST /api/v1/auth/logout", handler.Logout)

	protected.HandleFunc("GET /api/v1/accounts", handler.ListAccounts)
	protected.HandleFunc("POST /api/v1/accounts", handler.CreateAccount)
	protected.HandleFunc("GET /api/v1/accounts/{id}", handler.GetAccount)
	protected.HandleFunc("PATCH /api/v1/accounts/{id}", handler.UpdateAccount)
	protected.HandleFunc("DELETE /api/v1/accounts/{id}", handler.ArchiveAccount)
	protected.HandleFunc("POST /api/v1/accounts/{id}/restore", handler.RestoreAccount)
	protected.HandleFunc("GET /api/v1/balances", handler.Balances)

	protected.HandleFunc("GET /api/v1/categories", handler.ListCategories)
	protected.HandleFunc("POST /api/v1/categories", handler.CreateCategory)
	protected.HandleFunc("PATCH /api/v1/categories/{id}", handler.UpdateCategory)
	protected.HandleFunc("DELETE /api/v1/categories/{id}", handler.ArchiveCategory)

	protected.HandleFunc("GET /api/v1/transactions", handler.ListTransactions)
	protected.HandleFunc("GET /api/v1/transactions/{id}", handler.GetTransaction)
	protected.HandleFunc("PATCH /api/v1/transactions/{id}", handler.UpdateTransaction)
	protected.HandleFunc("POST /api/v1/transactions/{id}/reversals", handler.ReverseTransaction)
	protected.HandleFunc("POST /api/v1/incomes", handler.CreateIncome)
	protected.HandleFunc("POST /api/v1/expenses", handler.CreateExpense)

	protected.HandleFunc("GET /api/v1/transfers", handler.ListTransfers)
	protected.HandleFunc("POST /api/v1/transfers", handler.CreateTransfer)
	protected.HandleFunc("POST /api/v1/transfers/{id}/reversal", handler.ReverseTransfer)

	protected.HandleFunc("GET /api/v1/debts", handler.ListDebts)
	protected.HandleFunc("POST /api/v1/debts", handler.CreateDebt)
	protected.HandleFunc("GET /api/v1/debts/{id}", handler.GetDebt)
	protected.HandleFunc("GET /api/v1/debts/{id}/installments", handler.ListDebtInstallments)
	protected.HandleFunc("PATCH /api/v1/debts/{id}", handler.UpdateDebt)
	protected.HandleFunc("DELETE /api/v1/debts/{id}", handler.DeleteDebt)
	protected.HandleFunc("GET /api/v1/debts/{id}/payments", handler.ListPayments)
	protected.HandleFunc("POST /api/v1/debts/{id}/payments", handler.CreatePayment)
	protected.HandleFunc("POST /api/v1/payments/{id}/reversal", handler.ReversePayment)

	protected.HandleFunc("GET /api/v1/planned-cash-flows", handler.ListPlanned)
	protected.HandleFunc("POST /api/v1/planned-cash-flows", handler.CreatePlanned)
	protected.HandleFunc("PATCH /api/v1/planned-cash-flows/{id}", handler.UpdatePlanned)
	protected.HandleFunc("DELETE /api/v1/planned-cash-flows/{id}", handler.DeletePlanned)
	protected.HandleFunc("POST /api/v1/planned-cash-flows/{id}/realization", handler.RealizePlanned)

	protected.HandleFunc("GET /api/v1/reports/spending", handler.SpendingReport)
	protected.HandleFunc("GET /api/v1/reports/monthly-average", handler.MonthlyAverage)
	protected.HandleFunc("POST /api/v1/reports/projections", handler.Projection)
	protected.HandleFunc("GET /api/v1/exports/transactions", handler.ExportTransactions)

	protected.HandleFunc("GET /api/v1/admin/users", handler.AdminListUsers)
	protected.HandleFunc("POST /api/v1/admin/users", handler.AdminCreateUser)
	protected.HandleFunc("POST /api/v1/admin/users/{userId}/accounts", handler.AdminCreateAccount)

	public.Handle("/api/v1/", authenticate(service, protected))
	return cors(corsOrigin, recoverer(logger, requestLogger(logger, requestIdentifier(public))))
}

func authenticate(service *service.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token == "" {
			if cookie, err := r.Cookie("financial_session"); err == nil {
				token = cookie.Value
			}
		}
		if token == "" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			http.Error(w, `{"code":"unauthorized","message":"Token ausente."}`, http.StatusUnauthorized)
			return
		}
		userID, err := service.ParseToken(r.Context(), token)
		if err != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			http.Error(w, `{"code":"unauthorized","message":"Token invalido ou expirado."}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(controller.WithUserID(r.Context(), userID)))
	})
}

func requestIdentifier(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			buffer := make([]byte, 12)
			_, _ = rand.Read(buffer)
			id = hex.EncodeToString(buffer)
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(controller.WithRequestID(r.Context(), id)))
	})
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http_request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(started).Milliseconds())
	})
}

func recoverer(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic", "error", recovered)
				http.Error(w, `{"code":"internal_error","message":"Erro interno."}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func cors(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
