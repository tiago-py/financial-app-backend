BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name varchar(120) NOT NULL,
    email varchar(255) NOT NULL,
    password_hash text NOT NULL,
    timezone varchar(64) NOT NULL DEFAULT 'America/Sao_Paulo',
    currency char(3) NOT NULL DEFAULT 'BRL' CHECK (currency = 'BRL'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_unique UNIQUE (email)
);

CREATE TABLE IF NOT EXISTS accounts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name varchar(120) NOT NULL,
    institution varchar(120),
    type varchar(40) NOT NULL DEFAULT 'checking',
    currency char(3) NOT NULL DEFAULT 'BRL' CHECK (currency = 'BRL'),
    opening_balance_cents bigint NOT NULL DEFAULT 0,
    opened_on date NOT NULL DEFAULT CURRENT_DATE,
    color varchar(16),
    archived_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS accounts_owner_idx ON accounts(owner_id, archived_at, created_at);

CREATE TABLE IF NOT EXISTS categories (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name varchar(100) NOT NULL,
    purpose varchar(16) NOT NULL CHECK (purpose IN ('income', 'expense')),
    color varchar(16),
    archived_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT categories_owner_name_purpose_unique UNIQUE (owner_id, name, purpose)
);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id uuid NOT NULL REFERENCES accounts(id),
    category_id uuid REFERENCES categories(id),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    currency char(3) NOT NULL DEFAULT 'BRL' CHECK (currency = 'BRL'),
    direction varchar(8) NOT NULL CHECK (direction IN ('in', 'out')),
    kind varchar(24) NOT NULL CHECK (kind IN ('income', 'expense', 'transfer', 'debt_payment', 'reversal')),
    occurred_on date NOT NULL,
    description varchar(240) NOT NULL,
    origin_type varchar(24),
    origin_id uuid,
    reversal_of_id uuid REFERENCES ledger_entries(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ledger_reversal_unique UNIQUE (reversal_of_id)
);
CREATE INDEX IF NOT EXISTS ledger_owner_date_idx ON ledger_entries(owner_id, occurred_on DESC, id DESC);
CREATE INDEX IF NOT EXISTS ledger_account_date_idx ON ledger_entries(owner_id, account_id, occurred_on DESC);

CREATE TABLE IF NOT EXISTS transfers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_account_id uuid NOT NULL REFERENCES accounts(id),
    to_account_id uuid NOT NULL REFERENCES accounts(id),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    currency char(3) NOT NULL DEFAULT 'BRL' CHECK (currency = 'BRL'),
    occurred_on date NOT NULL,
    description varchar(240) NOT NULL DEFAULT 'Transferencia entre contas',
    out_entry_id uuid NOT NULL UNIQUE REFERENCES ledger_entries(id),
    in_entry_id uuid NOT NULL UNIQUE REFERENCES ledger_entries(id),
    reversed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (from_account_id <> to_account_id)
);

CREATE TABLE IF NOT EXISTS debts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    description varchar(180) NOT NULL,
    creditor varchar(120),
    principal_cents bigint NOT NULL CHECK (principal_cents > 0),
    currency char(3) NOT NULL DEFAULT 'BRL' CHECK (currency = 'BRL'),
    due_date date,
    status varchar(16) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paid', 'cancelled', 'archived')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS debts_owner_status_due_idx ON debts(owner_id, status, due_date);

CREATE TABLE IF NOT EXISTS debt_payments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    debt_id uuid NOT NULL REFERENCES debts(id),
    account_id uuid NOT NULL REFERENCES accounts(id),
    ledger_entry_id uuid NOT NULL UNIQUE REFERENCES ledger_entries(id),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    paid_on date NOT NULL,
    reversed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS payments_debt_date_idx ON debt_payments(debt_id, paid_on DESC);

CREATE TABLE IF NOT EXISTS planned_cash_flows (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id uuid REFERENCES accounts(id),
    category_id uuid REFERENCES categories(id),
    direction varchar(8) NOT NULL CHECK (direction IN ('in', 'out')),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    currency char(3) NOT NULL DEFAULT 'BRL' CHECK (currency = 'BRL'),
    expected_on date NOT NULL,
    description varchar(240) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'planned' CHECK (status IN ('planned', 'realized', 'cancelled')),
    realized_entry_id uuid REFERENCES ledger_entries(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS planned_owner_date_idx ON planned_cash_flows(owner_id, status, expected_on);

CREATE TABLE IF NOT EXISTS idempotency_records (
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation varchar(60) NOT NULL,
    key varchar(160) NOT NULL,
    request_hash char(64) NOT NULL,
    resource_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (owner_id, operation, key)
);

COMMIT;
