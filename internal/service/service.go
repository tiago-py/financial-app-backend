package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"financial-app-backend/internal/auth"
	"financial-app-backend/internal/cache"
	"financial-app-backend/internal/model"
	"financial-app-backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	store  *repository.Store
	cache  *cache.Cache
	tokens *auth.Manager
}

func New(store *repository.Store, cache *cache.Cache, tokens *auth.Manager) *Service {
	return &Service{store: store, cache: cache, tokens: tokens}
}

func (s *Service) Ping(ctx context.Context) error { return s.store.Ping(ctx) }
func (s *Service) ParseToken(ctx context.Context, token string) (string, error) {
	userID, version, err := s.tokens.ParseVersion(token)
	if err != nil {
		return "", model.ErrUnauthorized
	}
	current, err := s.store.SessionVersion(ctx, userID)
	if err != nil || current != version {
		return "", model.ErrUnauthorized
	}
	return userID, nil
}

func validateUserInput(name, email, password string) error {
	if err := model.ValidateRequired(name, "name", 120); err != nil {
		return err
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return model.ValidationError{Fields: []model.FieldError{{Field: "email", Message: "email invalido"}}}
	}
	if len(password) < 8 || len(password) > 72 {
		return model.ValidationError{Fields: []model.FieldError{{Field: "password", Message: "use entre 8 e 72 caracteres"}}}
	}
	return nil
}

func (s *Service) BootstrapAdmin(ctx context.Context, name, email, password string) error {
	if err := validateUserInput(name, email, password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.store.UpsertAdmin(ctx, name, email, string(hash))
}

func (s *Service) AdminCreateUser(ctx context.Context, actorID, name, email, password string) (model.User, error) {
	if err := s.requireAdmin(ctx, actorID); err != nil {
		return model.User{}, err
	}
	if err := validateUserInput(name, email, password); err != nil {
		return model.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}
	return s.store.CreateUser(ctx, name, email, string(hash), "user")
}

func (s *Service) AdminListUsers(ctx context.Context, actorID string) ([]model.User, error) {
	if err := s.requireAdmin(ctx, actorID); err != nil {
		return nil, err
	}
	return s.store.ListUsers(ctx)
}

func (s *Service) requireAdmin(ctx context.Context, actorID string) error {
	user, err := s.store.UserByID(ctx, actorID)
	if err != nil {
		return err
	}
	if user.Role != "admin" {
		return model.ErrForbidden
	}
	return nil
}

func (s *Service) Login(ctx context.Context, email, password string) (model.User, string, error) {
	user, hash, version, err := s.store.UserByEmail(ctx, email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return model.User{}, "", model.ErrUnauthorized
	}
	token, err := s.tokens.IssueVersion(user.ID, version)
	return user, token, err
}

func (s *Service) Me(ctx context.Context, ownerID string) (model.User, error) {
	return s.store.UserByID(ctx, ownerID)
}

func (s *Service) UpdateMe(ctx context.Context, ownerID, name, timezone, currency string) (model.User, error) {
	if currency != "" && strings.ToUpper(currency) != "BRL" {
		return model.User{}, model.ErrInvalid
	}
	user, err := s.store.UpdateUser(ctx, ownerID, name, timezone, currency)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return user, err
}

func (s *Service) ListAccounts(ctx context.Context, ownerID string, archived bool) ([]model.Account, error) {
	return s.store.ListAccounts(ctx, ownerID, archived)
}

func (s *Service) GetAccount(ctx context.Context, ownerID, id string) (model.Account, error) {
	return s.store.GetAccount(ctx, ownerID, id)
}

func (s *Service) CreateAccount(ctx context.Context, ownerID string, input repository.AccountInput) (model.Account, error) {
	if err := model.ValidateRequired(input.Name, "name", 120); err != nil {
		return model.Account{}, err
	}
	if err := model.ValidateDate(input.OpenedOn, "openedOn"); err != nil {
		return model.Account{}, err
	}
	if len(input.Institution) > 120 || len(input.Type) > 40 || len(input.Color) > 16 {
		return model.Account{}, model.ErrInvalid
	}
	if input.OpeningBalanceCents < -9007199254740991 || input.OpeningBalanceCents > 9007199254740991 {
		return model.Account{}, model.ValidationError{Fields: []model.FieldError{{Field: "openingBalanceCents", Message: "saldo fora do limite permitido"}}}
	}
	account, err := s.store.CreateAccount(ctx, ownerID, input)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return account, err
}

func (s *Service) AdminCreateAccount(ctx context.Context, actorID, ownerID string, input repository.AccountInput) (model.Account, error) {
	if err := s.requireAdmin(ctx, actorID); err != nil {
		return model.Account{}, err
	}
	if _, err := s.store.UserByID(ctx, ownerID); err != nil {
		return model.Account{}, err
	}
	return s.CreateAccount(ctx, ownerID, input)
}

func (s *Service) UpdateAccount(ctx context.Context, ownerID, id string, input repository.AccountInput) (model.Account, error) {
	account, err := s.store.UpdateAccount(ctx, ownerID, id, input)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return account, err
}

func (s *Service) ArchiveAccount(ctx context.Context, ownerID, id string) (model.Account, error) {
	account, err := s.store.ArchiveAccount(ctx, ownerID, id)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return account, err
}

func (s *Service) RestoreAccount(ctx context.Context, ownerID, id string) (model.Account, error) {
	account, err := s.store.RestoreAccount(ctx, ownerID, id)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return account, err
}

func (s *Service) ListCategories(ctx context.Context, ownerID, purpose string, archived bool) ([]model.Category, error) {
	return s.store.ListCategories(ctx, ownerID, purpose, archived)
}

func (s *Service) CreateCategory(ctx context.Context, ownerID string, input repository.CategoryInput) (model.Category, error) {
	if err := model.ValidateRequired(input.Name, "name", 100); err != nil {
		return model.Category{}, err
	}
	if input.Purpose != "income" && input.Purpose != "expense" {
		return model.Category{}, model.ErrInvalid
	}
	category, err := s.store.CreateCategory(ctx, ownerID, input)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return category, err
}

func (s *Service) UpdateCategory(ctx context.Context, ownerID, id string, input repository.CategoryInput) (model.Category, error) {
	if input.Purpose != "" && input.Purpose != "income" && input.Purpose != "expense" {
		return model.Category{}, model.ErrInvalid
	}
	category, err := s.store.UpdateCategory(ctx, ownerID, id, input)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return category, err
}

func (s *Service) ArchiveCategory(ctx context.Context, ownerID, id string) (model.Category, error) {
	category, err := s.store.ArchiveCategory(ctx, ownerID, id)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return category, err
}

func (s *Service) ListTransactions(ctx context.Context, ownerID string, filter repository.TransactionFilter) ([]model.LedgerEntry, int, error) {
	return s.store.ListTransactions(ctx, ownerID, filter)
}

func (s *Service) ExportTransactions(ctx context.Context, ownerID string, filter repository.TransactionFilter) ([]model.LedgerEntry, error) {
	filter.Offset = 0
	filter.Limit = 100
	result := make([]model.LedgerEntry, 0)
	for {
		items, total, err := s.store.ListTransactions(ctx, ownerID, filter)
		if err != nil {
			return nil, err
		}
		result = append(result, items...)
		if len(result) >= total || len(items) == 0 {
			return result, nil
		}
		filter.Offset += len(items)
	}
}

func (s *Service) GetTransaction(ctx context.Context, ownerID, id string) (model.LedgerEntry, error) {
	return s.store.GetTransaction(ctx, ownerID, id)
}

func (s *Service) CreateTransaction(ctx context.Context, ownerID string, input repository.EntryInput) (model.LedgerEntry, error) {
	if input.Kind != "income" && input.Kind != "expense" {
		return model.LedgerEntry{}, model.ErrInvalid
	}
	expectedDirection := "in"
	if input.Kind == "expense" {
		expectedDirection = "out"
	}
	input.Direction = expectedDirection
	if err := validateEntry(input); err != nil {
		return model.LedgerEntry{}, err
	}
	entry, err := s.store.CreateEntry(ctx, ownerID, input)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return entry, err
}

func (s *Service) UpdateTransaction(ctx context.Context, ownerID, id, description string, categoryID *string) (model.LedgerEntry, error) {
	entry, err := s.store.UpdateTransaction(ctx, ownerID, id, description, categoryID)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return entry, err
}

func (s *Service) ReverseTransaction(ctx context.Context, ownerID, id, reason, occurredOn string) (model.LedgerEntry, error) {
	if err := model.ValidateRequired(reason, "reason", 240); err != nil {
		return model.LedgerEntry{}, err
	}
	if err := model.ValidateDate(occurredOn, "occurredOn"); err != nil {
		return model.LedgerEntry{}, err
	}
	entry, err := s.store.ReverseTransaction(ctx, ownerID, id, reason, occurredOn)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return entry, err
}

func validateEntry(input repository.EntryInput) error {
	if err := model.ValidateRequired(input.AccountID, "accountId", 64); err != nil {
		return err
	}
	if err := model.ValidatePositiveCents(input.AmountCents, "amountCents"); err != nil {
		return err
	}
	if err := model.ValidateDate(input.OccurredOn, "occurredOn"); err != nil {
		return err
	}
	return model.ValidateRequired(input.Description, "description", 240)
}

func (s *Service) ListTransfers(ctx context.Context, ownerID string) ([]model.Transfer, error) {
	return s.store.ListTransfers(ctx, ownerID)
}

func (s *Service) CreateTransfer(ctx context.Context, ownerID string, input repository.TransferInput, key string) (model.Transfer, error) {
	if strings.TrimSpace(key) == "" || len(key) > 160 {
		return model.Transfer{}, model.ErrIdempotencyKey
	}
	if input.FromAccountID == input.ToAccountID {
		return model.Transfer{}, model.ErrInvalid
	}
	if err := model.ValidatePositiveCents(input.AmountCents, "amountCents"); err != nil {
		return model.Transfer{}, err
	}
	if err := model.ValidateDate(input.OccurredOn, "occurredOn"); err != nil {
		return model.Transfer{}, err
	}
	if strings.TrimSpace(input.Description) == "" {
		input.Description = "Transferencia entre contas"
	}
	item, err := s.store.CreateTransfer(ctx, ownerID, input, key, requestHash(input))
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return item, err
}

func (s *Service) ReverseTransfer(ctx context.Context, ownerID, id, reason, occurredOn string) (model.Transfer, error) {
	if err := model.ValidateDate(occurredOn, "occurredOn"); err != nil {
		return model.Transfer{}, err
	}
	item, err := s.store.ReverseTransfer(ctx, ownerID, id, reason, occurredOn)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return item, err
}

func (s *Service) ListDebts(ctx context.Context, ownerID, status string) ([]model.Debt, error) {
	return s.store.ListDebts(ctx, ownerID, status)
}

func (s *Service) GetDebt(ctx context.Context, ownerID, id string) (model.Debt, error) {
	return s.store.GetDebt(ctx, ownerID, id)
}

func (s *Service) CreateDebt(ctx context.Context, ownerID string, input repository.DebtInput) (model.Debt, error) {
	if err := model.ValidateRequired(input.Description, "description", 180); err != nil {
		return model.Debt{}, err
	}
	if err := model.ValidatePositiveCents(input.PrincipalCents, "principalCents"); err != nil {
		return model.Debt{}, err
	}
	if input.DueDate != nil {
		if err := model.ValidateDate(*input.DueDate, "dueDate"); err != nil {
			return model.Debt{}, err
		}
	}
	debt, err := s.store.CreateDebt(ctx, ownerID, input)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return debt, err
}

func (s *Service) UpdateDebt(ctx context.Context, ownerID, id string, input repository.DebtInput) (model.Debt, error) {
	if input.Status != "" && input.Status != "active" && input.Status != "cancelled" && input.Status != "archived" {
		return model.Debt{}, model.ErrInvalid
	}
	if input.DueDate != nil {
		if err := model.ValidateDate(*input.DueDate, "dueDate"); err != nil {
			return model.Debt{}, err
		}
	}
	debt, err := s.store.UpdateDebt(ctx, ownerID, id, input)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return debt, err
}

func (s *Service) DeleteDebt(ctx context.Context, ownerID, id string) error {
	err := s.store.DeleteDebt(ctx, ownerID, id)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return err
}

func (s *Service) ListPayments(ctx context.Context, ownerID, debtID string) ([]model.DebtPayment, error) {
	return s.store.ListPayments(ctx, ownerID, debtID)
}

func (s *Service) CreatePayment(ctx context.Context, ownerID, debtID, accountID string, amount int64, paidOn, key string) (model.DebtPayment, error) {
	if strings.TrimSpace(key) == "" || len(key) > 160 {
		return model.DebtPayment{}, model.ErrIdempotencyKey
	}
	if err := model.ValidatePositiveCents(amount, "amountCents"); err != nil {
		return model.DebtPayment{}, err
	}
	if err := model.ValidateDate(paidOn, "paidOn"); err != nil {
		return model.DebtPayment{}, err
	}
	input := struct {
		DebtID    string
		AccountID string
		Amount    int64
		PaidOn    string
	}{debtID, accountID, amount, paidOn}
	item, err := s.store.CreatePayment(ctx, ownerID, debtID, accountID, amount, paidOn, key, requestHash(input))
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return item, err
}

func (s *Service) ReversePayment(ctx context.Context, ownerID, id, reason, occurredOn string) (model.DebtPayment, error) {
	if err := model.ValidateDate(occurredOn, "occurredOn"); err != nil {
		return model.DebtPayment{}, err
	}
	item, err := s.store.ReversePayment(ctx, ownerID, id, reason, occurredOn)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return item, err
}

func (s *Service) ListPlanned(ctx context.Context, ownerID, status, from, to string) ([]model.PlannedCashFlow, error) {
	return s.store.ListPlanned(ctx, ownerID, status, from, to)
}

func (s *Service) CreatePlanned(ctx context.Context, ownerID string, input repository.PlannedInput) (model.PlannedCashFlow, error) {
	if err := validatePlanned(input); err != nil {
		return model.PlannedCashFlow{}, err
	}
	item, err := s.store.CreatePlanned(ctx, ownerID, input)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return item, err
}

func (s *Service) UpdatePlanned(ctx context.Context, ownerID, id string, input repository.PlannedInput) (model.PlannedCashFlow, error) {
	if input.Direction != "" {
		if err := model.ValidateDirection(input.Direction); err != nil {
			return model.PlannedCashFlow{}, err
		}
	}
	if input.AmountCents < 0 {
		return model.PlannedCashFlow{}, model.ErrInvalid
	}
	if input.ExpectedOn != "" {
		if err := model.ValidateDate(input.ExpectedOn, "expectedOn"); err != nil {
			return model.PlannedCashFlow{}, err
		}
	}
	if input.Status != "" && input.Status != "planned" && input.Status != "cancelled" {
		return model.PlannedCashFlow{}, model.ErrInvalid
	}
	item, err := s.store.UpdatePlanned(ctx, ownerID, id, input)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return item, err
}

func (s *Service) DeletePlanned(ctx context.Context, ownerID, id string) error {
	err := s.store.DeletePlanned(ctx, ownerID, id)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return err
}

func (s *Service) RealizePlanned(ctx context.Context, ownerID, id, accountID, occurredOn string) (model.PlannedCashFlow, error) {
	if err := model.ValidateDate(occurredOn, "occurredOn"); err != nil {
		return model.PlannedCashFlow{}, err
	}
	item, err := s.store.RealizePlanned(ctx, ownerID, id, accountID, occurredOn)
	if err == nil {
		s.cache.InvalidateUser(ctx, ownerID)
	}
	return item, err
}

func validatePlanned(input repository.PlannedInput) error {
	if err := model.ValidateDirection(input.Direction); err != nil {
		return err
	}
	if err := model.ValidatePositiveCents(input.AmountCents, "amountCents"); err != nil {
		return err
	}
	if err := model.ValidateDate(input.ExpectedOn, "expectedOn"); err != nil {
		return err
	}
	return model.ValidateRequired(input.Description, "description", 240)
}

func (s *Service) Balances(ctx context.Context, ownerID string) (model.Balances, error) {
	key := "user:" + ownerID + ":balances"
	var result model.Balances
	if found, _ := s.cache.GetJSON(ctx, key, &result); found {
		return result, nil
	}
	result, err := s.store.Balances(ctx, ownerID)
	if err == nil {
		_ = s.cache.SetJSON(ctx, key, result, 2*time.Minute)
	}
	return result, err
}

func (s *Service) Spending(ctx context.Context, ownerID, from, to string) (model.SpendingReport, error) {
	if err := model.ValidateDate(from, "from"); err != nil {
		return model.SpendingReport{}, err
	}
	if err := model.ValidateDate(to, "to"); err != nil {
		return model.SpendingReport{}, err
	}
	key := fmt.Sprintf("user:%s:spending:%s:%s", ownerID, from, to)
	var result model.SpendingReport
	if found, _ := s.cache.GetJSON(ctx, key, &result); found {
		return result, nil
	}
	result, err := s.store.Spending(ctx, ownerID, from, to)
	if err == nil {
		_ = s.cache.SetJSON(ctx, key, result, 5*time.Minute)
	}
	return result, err
}

func (s *Service) MonthlyAverage(ctx context.Context, ownerID, from, to string) (int64, int, error) {
	return s.store.MonthlyAverage(ctx, ownerID, from, to)
}

func (s *Service) Projection(ctx context.Context, ownerID, baseDate string, months int) ([]model.ProjectionPoint, error) {
	date, err := time.Parse("2006-01-02", baseDate)
	if err != nil || months < 1 || months > 24 {
		return nil, model.ErrInvalid
	}
	return s.store.Projection(ctx, ownerID, date, months)
}

func requestHash(value any) string {
	encoded, _ := json.Marshal(value)
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:])
}
