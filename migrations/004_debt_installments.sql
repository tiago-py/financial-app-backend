CREATE TABLE IF NOT EXISTS debt_installments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    debt_id uuid NOT NULL REFERENCES debts(id) ON DELETE CASCADE,
    installment_number integer NOT NULL CHECK (installment_number > 0),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    due_date date NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (debt_id, installment_number)
);

CREATE INDEX IF NOT EXISTS debt_installments_debt_due_idx
    ON debt_installments(debt_id, due_date);
