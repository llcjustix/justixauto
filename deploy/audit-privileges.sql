-- OFFLINE PROVISIONING TEMPLATE ONLY. Nothing invokes this file automatically.
-- This file is not authorization to execute against any environment, especially
-- production. No role, credential, login mapping or deployment secret is created.
-- A separately approved rollout must supply EXISTING, distinct runtime/owner
-- roles, protected migration credentials, a compatible migration-47 baseline,
-- environment/backup/rollback scope, and review other SECURITY DEFINER helpers.
-- The runtime connection must actually use the supplied runtime role. Installing
-- this template does not change the application's current owner credentials.
--
-- Manual invocation shape, after that separate approval (replace both values):
-- psql -X --set=runtime_role='<existing runtime>' \
--   --set=owner_role='<existing audit owner>' --file=deploy/audit-privileges.sql
-- Supply the connection separately; never embed credentials here. Use a trusted
-- provisioning connection with authority to revoke PUBLIC schema CREATE as well
-- as the audit ACLs. Revoking PUBLIC CREATE on public affects other users: review
-- that prerequisite before execution. This template intentionally rejects ALL
-- runtime memberships and object ownership, rather than attempting to repair an
-- unknown inherited/SET ROLE graph or transfer ownership. Review/remediate those
-- separately; it never guesses roles or applies blanket module grants.
--
-- The transaction commits ACL changes only if all effective checks pass. Any SQL
-- failure stops psql and leaves the transaction uncommitted (disconnect rolls it
-- back). An interactive session must ROLLBACK after an error; never COMMIT it.
-- Owner/superuser credentials can still replace guards/rewrite history. Retain
-- checkpoints externally; these ACLs cannot make unanchored history trusted.
\set ON_ERROR_STOP on
\if :{?runtime_role}
\else
  \echo 'Required psql variable missing: runtime_role (existing role name)'
  \quit 3
\endif
\if :{?owner_role}
\else
  \echo 'Required psql variable missing: owner_role (existing role name)'
  \quit 3
\endif

BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL standard_conforming_strings = on;
-- :'name' quotes values as SQL literals; :"name" below quotes identifiers.
SELECT pg_catalog.set_config('justix.audit_runtime_role', :'runtime_role', true),
       pg_catalog.set_config('justix.audit_owner_role', :'owner_role', true);

DO $provision$
DECLARE
    runtime_name text := pg_catalog.current_setting('justix.audit_runtime_role');
    owner_name text := pg_catalog.current_setting('justix.audit_owner_role');
    runtime_role pg_catalog.pg_roles%ROWTYPE;
    owner_id oid;
    sequence_id oid;
    object_id oid;
    object_name text;
    function_ids oid[] := ARRAY[
        pg_catalog.to_regprocedure('public.identity_audit_lock()')::oid,
        pg_catalog.to_regprocedure('public.identity_audit_append()')::oid,
        pg_catalog.to_regprocedure('public.identity_audit_head_event_required()')::oid,
        pg_catalog.to_regprocedure('public.identity_audit_reject_mutation()')::oid,
        pg_catalog.to_regprocedure('public.identity_audit_canonical_v1(uuid,public.identity_audit_events)')::oid,
        pg_catalog.to_regprocedure('public.identity_audit_boundary_v1(uuid,bigint,bigint,timestamptz)')::oid
    ];
    column_record record;
BEGIN
    SELECT * INTO runtime_role FROM pg_catalog.pg_roles WHERE rolname = runtime_name;
    SELECT oid INTO owner_id FROM pg_catalog.pg_roles WHERE rolname = owner_name;
    IF runtime_role.oid IS NULL OR owner_id IS NULL OR runtime_role.oid = owner_id THEN
        RAISE EXCEPTION 'Supply distinct existing runtime and audit-owner roles';
    END IF;
    IF runtime_role.rolsuper OR runtime_role.rolbypassrls OR runtime_role.rolcreaterole
        OR runtime_role.rolcreatedb OR runtime_role.rolreplication OR runtime_role.rolinherit THEN
        RAISE EXCEPTION 'Runtime requires NOSUPERUSER NOBYPASSRLS NOCREATEROLE NOCREATEDB NOREPLICATION NOINHERIT';
    END IF;
    -- Rejecting every direct membership also closes every transitive inherited
    -- or SET ROLE route, including NOINHERIT roles and membership admin options.
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members WHERE member = runtime_role.oid) THEN
        RAISE EXCEPTION 'Runtime must have no role memberships (including SET ROLE routes)';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_database
            WHERE datname = pg_catalog.current_database() AND datdba = runtime_role.oid)
        OR EXISTS (SELECT 1 FROM pg_catalog.pg_namespace WHERE nspowner = runtime_role.oid)
        OR EXISTS (SELECT 1 FROM pg_catalog.pg_class WHERE relowner = runtime_role.oid)
        OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc WHERE proowner = runtime_role.oid)
        OR EXISTS (SELECT 1 FROM pg_catalog.pg_type WHERE typowner = runtime_role.oid) THEN
        RAISE EXCEPTION 'Runtime must not own the database, schemas, relations, routines or types';
    END IF;
    IF pg_catalog.has_database_privilege(runtime_role.oid, pg_catalog.current_database(), 'CREATE')
        OR pg_catalog.has_parameter_privilege(runtime_role.oid, 'session_replication_role', 'SET')
        OR pg_catalog.has_parameter_privilege(runtime_role.oid, 'session_replication_role', 'ALTER SYSTEM') THEN
        RAISE EXCEPTION 'Runtime must not create schemas or bypass triggers via session_replication_role';
    END IF;

    sequence_id := pg_catalog.pg_get_serial_sequence('public.identity_audit_events', 'seq')::pg_catalog.regclass::oid;
    IF sequence_id IS NULL OR pg_catalog.array_position(function_ids, NULL) IS NOT NULL THEN
        RAISE EXCEPTION 'Migration 47 audit sequence/functions must exist with exact signatures';
    END IF;
    FOREACH object_id IN ARRAY ARRAY[
        'public.identity_audit_events'::pg_catalog.regclass::oid,
        'public.identity_audit_chain'::pg_catalog.regclass::oid, sequence_id
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class
                WHERE oid = object_id AND relowner = owner_id
                    AND relkind = CASE WHEN object_id = sequence_id THEN 'S'::"char" ELSE 'r'::"char" END) THEN
            RAISE EXCEPTION 'Audit tables/sequence must belong to supplied owner with expected object kinds';
        END IF;
    END LOOP;
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_proc AS p
            JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
            WHERE n.nspname = 'public' AND p.proname LIKE 'identity\_audit\_%' ESCAPE '\'
                AND NOT (p.oid = ANY(function_ids))) THEN
        RAISE EXCEPTION 'Unexpected audit routine/overload requires a new reviewed privilege contract';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_proc WHERE oid = ANY(function_ids)
            AND (proowner <> owner_id OR proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']
                OR prosecdef IS DISTINCT FROM (proname IN (
                    'identity_audit_lock', 'identity_audit_append', 'identity_audit_head_event_required')))) THEN
        RAISE EXCEPTION 'Audit function ownership, SECURITY DEFINER or fixed search_path differs from migration 47';
    END IF;
    IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger
            WHERE (tgrelid, tgname, tgfoid) IN (
                ('public.identity_audit_events'::pg_catalog.regclass, 'identity_audit_append', function_ids[2]),
                ('public.identity_audit_events'::pg_catalog.regclass, 'identity_audit_reject_mutation', function_ids[4]),
                ('public.identity_audit_events'::pg_catalog.regclass, 'identity_audit_reject_truncate', function_ids[4]),
                ('public.identity_audit_chain'::pg_catalog.regclass, 'identity_audit_head_event_required', function_ids[3]))
                AND tgenabled IN ('O', 'A')) <> 4 THEN
        RAISE EXCEPTION 'All four migration-47 audit triggers must be installed and enabled';
    END IF;

    -- Remove both table and column ACLs: table REVOKE alone leaves old column
    -- UPDATE/REFERENCES grants intact. Never mutate rows or the chain head here.
    FOREACH object_id IN ARRAY ARRAY[
        'public.identity_audit_events'::pg_catalog.regclass::oid,
        'public.identity_audit_chain'::pg_catalog.regclass::oid
    ] LOOP
        SELECT pg_catalog.format('%I.%I', n.nspname, c.relname) INTO object_name
            FROM pg_catalog.pg_class AS c JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
            WHERE c.oid = object_id;
        EXECUTE pg_catalog.format('REVOKE ALL PRIVILEGES ON TABLE %s FROM PUBLIC, %I', object_name, runtime_name);
        FOR column_record IN SELECT attname FROM pg_catalog.pg_attribute
            WHERE attrelid = object_id AND attnum > 0 AND NOT attisdropped LOOP
            EXECUTE pg_catalog.format('REVOKE ALL PRIVILEGES (%I) ON TABLE %s FROM PUBLIC, %I',
                column_record.attname, object_name, runtime_name);
        END LOOP;
    END LOOP;
    SELECT pg_catalog.format('%I.%I', n.nspname, c.relname) INTO object_name
        FROM pg_catalog.pg_class AS c JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
        WHERE c.oid = sequence_id;
    EXECUTE pg_catalog.format('REVOKE ALL PRIVILEGES ON SEQUENCE %s FROM PUBLIC, %I', object_name, runtime_name);
    EXECUTE pg_catalog.format('GRANT USAGE, SELECT ON SEQUENCE %s TO %I', object_name, runtime_name);
    FOREACH object_id IN ARRAY function_ids LOOP
        -- regprocedure renders schema/argument identifiers safely under the
        -- fixed search path; role identifiers use format %I, never raw values.
        EXECUTE pg_catalog.format('REVOKE ALL PRIVILEGES ON FUNCTION %s FROM PUBLIC, %I',
            object_id::pg_catalog.regprocedure, runtime_name);
    END LOOP;
END;
$provision$;

REVOKE CREATE ON SCHEMA public FROM PUBLIC, :"runtime_role";
GRANT USAGE ON SCHEMA public TO :"runtime_role";
-- SELECT includes INSERT ... RETURNING and forensic reads. Sequence UPDATE is
-- deliberately absent (no setval); only the trigger can advance the head.
GRANT SELECT, INSERT ON TABLE public.identity_audit_events TO :"runtime_role";
GRANT SELECT ON TABLE public.identity_audit_chain TO :"runtime_role";
GRANT EXECUTE ON FUNCTION public.identity_audit_lock() TO :"runtime_role";
GRANT EXECUTE ON FUNCTION public.identity_audit_canonical_v1(uuid, public.identity_audit_events) TO :"runtime_role";
GRANT EXECUTE ON FUNCTION public.identity_audit_boundary_v1(uuid, bigint, bigint, timestamptz) TO :"runtime_role";

DO $verify$
DECLARE
    runtime_id oid := (SELECT oid FROM pg_catalog.pg_roles
        WHERE rolname = pg_catalog.current_setting('justix.audit_runtime_role'));
    events_id oid := 'public.identity_audit_events'::pg_catalog.regclass::oid;
    head_id oid := 'public.identity_audit_chain'::pg_catalog.regclass::oid;
    sequence_id oid := pg_catalog.pg_get_serial_sequence('public.identity_audit_events', 'seq')::pg_catalog.regclass::oid;
    function_record record;
    required_execute boolean;
BEGIN
    -- has_* checks evaluate effective permissions, not just the grants above.
    -- Temporary namespaces are excluded: audit routines qualify every object,
    -- and their fixed search path places pg_temp after pg_catalog.
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_namespace
            WHERE nspname NOT LIKE 'pg\_temp\_%' ESCAPE '\'
                AND nspname NOT LIKE 'pg\_toast\_temp\_%' ESCAPE '\'
                AND pg_catalog.has_schema_privilege(runtime_id, oid, 'CREATE')) THEN
        RAISE EXCEPTION 'Runtime retains effective schema CREATE; resolve that grant separately';
    END IF;
    IF NOT pg_catalog.has_schema_privilege(runtime_id, 'public', 'USAGE')
        OR NOT pg_catalog.has_table_privilege(runtime_id, events_id, 'SELECT')
        OR NOT pg_catalog.has_table_privilege(runtime_id, events_id, 'INSERT')
        OR NOT pg_catalog.has_table_privilege(runtime_id, head_id, 'SELECT')
        OR NOT pg_catalog.has_sequence_privilege(runtime_id, sequence_id, 'USAGE')
        OR NOT pg_catalog.has_sequence_privilege(runtime_id, sequence_id, 'SELECT') THEN
        RAISE EXCEPTION 'Required effective audit read/append/RETURNING/sequence privileges are missing';
    END IF;
    IF pg_catalog.has_table_privilege(runtime_id, events_id, 'UPDATE,DELETE,TRUNCATE,TRIGGER,REFERENCES')
        OR pg_catalog.has_any_column_privilege(runtime_id, events_id, 'UPDATE,REFERENCES')
        OR pg_catalog.has_table_privilege(runtime_id, head_id, 'INSERT,UPDATE,DELETE,TRUNCATE,TRIGGER,REFERENCES')
        OR pg_catalog.has_any_column_privilege(runtime_id, head_id, 'INSERT,UPDATE,REFERENCES')
        OR pg_catalog.has_sequence_privilege(runtime_id, sequence_id, 'UPDATE') THEN
        RAISE EXCEPTION 'Runtime retains forbidden effective audit mutation privileges';
    END IF;
    -- Membership and ownership were rejected above, so remaining effective ACLs
    -- can only come from runtime or PUBLIC. Check all privilege names (including
    -- version-specific additions such as MAINTAIN), columns and grant options.
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_class AS c
            CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(c.relacl,
                pg_catalog.acldefault(CASE WHEN c.oid = sequence_id THEN 's'::"char" ELSE 'r'::"char" END, c.relowner))) AS acl
            WHERE c.oid IN (events_id, head_id, sequence_id) AND acl.grantee IN (0, runtime_id)
                AND (acl.is_grantable OR acl.grantee = 0 OR NOT (
                    (c.oid = events_id AND acl.privilege_type IN ('SELECT', 'INSERT'))
                    OR (c.oid = head_id AND acl.privilege_type = 'SELECT')
                    OR (c.oid = sequence_id AND acl.privilege_type IN ('USAGE', 'SELECT')))))
        OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute AS a
            CROSS JOIN LATERAL pg_catalog.aclexplode(a.attacl) AS acl
            WHERE a.attrelid IN (events_id, head_id) AND acl.grantee IN (0, runtime_id)) THEN
        RAISE EXCEPTION 'Unexpected audit ACL, PUBLIC permission, column grant or grant option remains';
    END IF;
    FOR function_record IN SELECT p.* FROM pg_catalog.pg_proc AS p
        JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
        WHERE n.nspname = 'public' AND p.proname LIKE 'identity\_audit\_%' ESCAPE '\' LOOP
        required_execute := function_record.proname IN (
            'identity_audit_lock', 'identity_audit_canonical_v1', 'identity_audit_boundary_v1');
        IF pg_catalog.has_function_privilege(runtime_id, function_record.oid, 'EXECUTE') IS DISTINCT FROM required_execute
            OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(COALESCE(function_record.proacl,
                    pg_catalog.acldefault('f', function_record.proowner))) AS acl
                WHERE acl.grantee IN (0, runtime_id) AND (acl.grantee = 0 OR acl.is_grantable)) THEN
            RAISE EXCEPTION 'Unexpected effective audit function privilege or grant option';
        END IF;
    END LOOP;
END;
$verify$;

COMMIT;
\echo 'Audit ACL transaction committed; verify deployment uses the supplied runtime role separately.'
