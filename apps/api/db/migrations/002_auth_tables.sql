-- +goose Up
-- +goose StatementBegin

-- AuthBoss-required columns on users
ALTER TABLE users ADD COLUMN password_hash     text;
ALTER TABLE users ADD COLUMN attempt_count     int         NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN last_attempt      timestamptz;
ALTER TABLE users ADD COLUMN locked_until       timestamptz;                         -- null = not locked
ALTER TABLE users ADD COLUMN recover_selector  text        UNIQUE;
ALTER TABLE users ADD COLUMN recover_verifier  text;
ALTER TABLE users ADD COLUMN recover_expiry    timestamptz;
ALTER TABLE users ADD COLUMN otps              text        NOT NULL DEFAULT '';  -- AuthBoss otp module: JSON list of hashed OTPs
ALTER TABLE users ADD COLUMN is_confirmed      boolean     NOT NULL DEFAULT false; -- phone verified

-- Opaque refresh tokens (long-lived, stored as SHA-256 hash)
CREATE TABLE refresh_tokens (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash  text        NOT NULL UNIQUE,
  device_type text,
  app         text        CHECK (app IN ('resident', 'guard')),
  expires_at  timestamptz NOT NULL DEFAULT now() + interval '30 days',
  revoked_at  timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens (token_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS refresh_tokens;
ALTER TABLE users DROP COLUMN IF EXISTS is_confirmed;
ALTER TABLE users DROP COLUMN IF EXISTS otps;
ALTER TABLE users DROP COLUMN IF EXISTS recover_expiry;
ALTER TABLE users DROP COLUMN IF EXISTS recover_verifier;
ALTER TABLE users DROP COLUMN IF EXISTS recover_selector;
ALTER TABLE users DROP COLUMN IF EXISTS locked_until;
ALTER TABLE users DROP COLUMN IF EXISTS last_attempt;
ALTER TABLE users DROP COLUMN IF EXISTS attempt_count;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
-- +goose StatementEnd
