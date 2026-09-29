package model

import "time"

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Timezone  string    `json:"timezone"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"createdAt"`
}

type Account struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Institution         *string    `json:"institution"`
	Type                string     `json:"type"`
	Currency            string     `json:"currency"`
	OpeningBalanceCents int64      `json:"openingBalanceCents"`
	OpenedOn            string     `json:"openedOn"`
	Color               *string    `json:"color"`
	ArchivedAt          *time.Time `json:"archivedAt"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type Category struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Purpose    string     `json:"purpose"`
	Color      *string    `json:"color"`
	ArchivedAt *time.Time `json:"archivedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type LedgerEntry struct {
	ID           string    `json:"id"`
	AccountID    string    `json:"accountId"`
	AccountName  string    `json:"accountName,omitempty"`
	CategoryID   *string   `json:"categoryId"`
	CategoryName *string   `json:"categoryName,omitempty"`
	AmountCents  int64     `json:"amountCents"`
	Currency     string    `json:"currency"`
	Direction    string    `json:"direction"`
	Kind         string    `json:"kind"`
	OccurredOn   string    `json:"occurredOn"`
	Description  string    `json:"description"`
	OriginType   *string   `json:"originType,omitempty"`
	OriginID     *string   `json:"originId,omitempty"`
	ReversalOfID *string   `json:"reversalOfId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Transfer struct {
	ID            string     `json:"id"`
	FromAccountID string     `json:"fromAccountId"`
	ToAccountID   string     `json:"toAccountId"`
	AmountCents   int64      `json:"amountCents"`
	Currency      string     `json:"currency"`
	OccurredOn    string     `json:"occurredOn"`
	Description   string     `json:"description"`
	ReversedAt    *time.Time `json:"reversedAt"`
	CreatedAt     time.Time  `json:"createdAt"`
}

type Debt struct {
	ID             string    `json:"id"`
	Description    string    `json:"description"`
	Creditor       *string   `json:"creditor"`
	PrincipalCents int64     `json:"principalCents"`
	PaidCents      int64     `json:"paidCents"`
	PendingCents   int64     `json:"pendingCents"`
	Currency       string    `json:"currency"`
	DueDate        *string   `json:"dueDate"`
	Status           string    `json:"status"`
	InstallmentCount int       `json:"installmentCount"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type DebtInstallment struct {
	ID          string    `json:"id"`
	DebtID      string    `json:"debtId"`
	Number      int       `json:"number"`
	AmountCents int64     `json:"amountCents"`
	DueDate     string    `json:"dueDate"`
	CreatedAt   time.Time `json:"createdAt"`
}

type DebtPayment struct {
	ID            string     `json:"id"`
	DebtID        string     `json:"debtId"`
	AccountID     string     `json:"accountId"`
	LedgerEntryID string     `json:"ledgerEntryId"`
	AmountCents   int64      `json:"amountCents"`
	PaidOn        string     `json:"paidOn"`
	ReversedAt    *time.Time `json:"reversedAt"`
	CreatedAt     time.Time  `json:"createdAt"`
}

type PlannedCashFlow struct {
	ID              string    `json:"id"`
	AccountID       *string   `json:"accountId"`
	CategoryID      *string   `json:"categoryId"`
	Direction       string    `json:"direction"`
	AmountCents     int64     `json:"amountCents"`
	Currency        string    `json:"currency"`
	ExpectedOn      string    `json:"expectedOn"`
	Description     string    `json:"description"`
	Status          string    `json:"status"`
	RealizedEntryID *string   `json:"realizedEntryId"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type AccountBalance struct {
	AccountID   string `json:"accountId"`
	AccountName string `json:"accountName"`
	AmountCents int64  `json:"amountCents"`
	Currency    string `json:"currency"`
}

type Balances struct {
	Accounts         []AccountBalance `json:"accounts"`
	TotalAmountCents int64            `json:"totalAmountCents"`
	Currency         string           `json:"currency"`
	AccountCount     int              `json:"accountCount"`
}

type SpendingCategory struct {
	CategoryID   *string `json:"categoryId"`
	CategoryName string  `json:"categoryName"`
	AmountCents  int64   `json:"amountCents"`
}

type SpendingReport struct {
	From             string             `json:"from"`
	To               string             `json:"to"`
	TotalAmountCents int64              `json:"totalAmountCents"`
	Currency         string             `json:"currency"`
	ByCategory       []SpendingCategory `json:"byCategory"`
}

type ProjectionPoint struct {
	Month                 string `json:"month"`
	ProjectedBalanceCents int64  `json:"projectedBalanceCents"`
	PlannedIncomeCents    int64  `json:"plannedIncomeCents"`
	PlannedOutflowCents   int64  `json:"plannedOutflowCents"`
}
