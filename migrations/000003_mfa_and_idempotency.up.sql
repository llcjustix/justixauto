-- TOTP multi-factor authentication. Secrets are AES-GCM encrypted by the app.
ALTER TABLE identity_users
    ADD COLUMN mfa_secret_enc   bytea,
    ADD COLUMN mfa_enabled_at   timestamptz,
    ADD COLUMN mfa_last_counter bigint NOT NULL DEFAULT 0; -- a TOTP step is accepted once

ALTER TABLE identity_sessions ADD COLUMN mfa_authenticated_at timestamptz;

CREATE TABLE identity_mfa_enrollments (
    id           uuid        PRIMARY KEY,
    user_id      uuid        NOT NULL REFERENCES identity_users ON DELETE CASCADE,
    secret_enc   bytea       NOT NULL,
    created_at   timestamptz NOT NULL,
    expires_at   timestamptz NOT NULL,
    confirmed_at timestamptz
);

CREATE TABLE identity_mfa_recovery_codes (
    user_id   uuid        NOT NULL REFERENCES identity_users ON DELETE CASCADE,
    code_hash bytea       NOT NULL,
    used_at   timestamptz,
    PRIMARY KEY (user_id, code_hash)
);

-- Password verified, second factor pending. Bound to the browser by a cookie.
CREATE TABLE identity_mfa_challenges (
    id          uuid        PRIMARY KEY,
    user_id     uuid        NOT NULL REFERENCES identity_users ON DELETE CASCADE,
    token_hash  bytea       NOT NULL,
    attempts    integer     NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz
);

-- Idempotency-Key ledger: a retried request replays the first response.
CREATE TABLE platform_idempotency_keys (
    actor_id      uuid        NOT NULL,
    key           uuid        NOT NULL,
    request_hash  bytea       NOT NULL,
    status        text        NOT NULL CHECK (status IN ('in_progress', 'completed')),
    response_code integer,
    response_etag text        NOT NULL DEFAULT '',
    response_body bytea,
    created_at    timestamptz NOT NULL,
    completed_at  timestamptz,
    PRIMARY KEY (actor_id, key)
);
