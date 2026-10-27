-- One-time reset of the dev database's pre-ADR-14 layout (user decision
-- 2026-10-27). Commit b512dee2 rewrote migrations 000001-000027 from
-- per-module schemas (commerce.orders) to prefixed public tables
-- (commerce_orders). A database migrated before that cannot apply 000028+.
-- This drops the old schemas and the migration history so `migrate up`
-- rebuilds everything. It deletes all data in those schemas. Once the old
-- layout is gone it does nothing, and it refuses any unexpected state.
BEGIN;
DO $reset$
DECLARE
    v_version bigint;
    v_dirty   boolean;
    v_extra   text;
BEGIN
    IF to_regclass('commerce.orders') IS NULL THEN
        RAISE NOTICE 'old schema layout not found; nothing to reset';
        RETURN;
    END IF;
    SELECT m.version, m.dirty INTO STRICT v_version, v_dirty FROM public.schema_migrations AS m;
    IF NOT ((v_version = 27 AND NOT v_dirty) OR (v_version = 28 AND v_dirty)) THEN
        RAISE EXCEPTION 'unexpected migration state version=% dirty=%; refusing to reset', v_version, v_dirty;
    END IF;
    SELECT string_agg(c.relname, ', ') INTO v_extra
        FROM pg_catalog.pg_class AS c JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
        WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p', 'v', 'm') AND c.relname <> 'schema_migrations';
    IF v_extra IS NOT NULL THEN
        RAISE EXCEPTION 'public schema already has tables (%); refusing to reset', v_extra;
    END IF;
    DROP SCHEMA IF EXISTS commerce, documents, financing, identity, insurance, inventory, platform, retail CASCADE;
    DROP TABLE public.schema_migrations;
    RAISE NOTICE 'old schema layout dropped; migrations will rebuild the database';
END
$reset$;
COMMIT;
