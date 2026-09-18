ALTER TABLE users
    ADD COLUMN IF NOT EXISTS role varchar(16) NOT NULL DEFAULT 'user';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'users_role_check'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'user'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS users_role_idx ON users(role, created_at);
