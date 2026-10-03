DROP TABLE IF EXISTS platform_idempotency_keys;
DROP TABLE IF EXISTS identity_mfa_challenges;
DROP TABLE IF EXISTS identity_mfa_recovery_codes;
DROP TABLE IF EXISTS identity_mfa_enrollments;
ALTER TABLE identity_sessions DROP COLUMN mfa_authenticated_at;
ALTER TABLE identity_users DROP COLUMN mfa_secret_enc, DROP COLUMN mfa_enabled_at, DROP COLUMN mfa_last_counter;
