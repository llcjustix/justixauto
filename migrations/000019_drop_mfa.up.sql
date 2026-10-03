-- Two-factor authentication is removed from the product (user decision 2026-09-24).
DROP TABLE IF EXISTS identity_mfa_challenges;
DROP TABLE IF EXISTS identity_mfa_recovery_codes;
DROP TABLE IF EXISTS identity_mfa_enrollments;
ALTER TABLE identity_sessions DROP COLUMN IF EXISTS mfa_authenticated_at;
ALTER TABLE identity_users
    DROP COLUMN IF EXISTS mfa_secret_enc,
    DROP COLUMN IF EXISTS mfa_enabled_at,
    DROP COLUMN IF EXISTS mfa_last_counter;
