BEGIN;

-- The ALTER lock fences the legacy boundary and remains held until the triggers
-- and head are installed. Existing rows receive NULL for every new event field.
ALTER TABLE public.identity_audit_events
    ALTER COLUMN resource_id DROP NOT NULL,
    ADD COLUMN actor_login text,
    ADD COLUMN actor_display_name text,
    ADD COLUMN outcome text CHECK (outcome IN ('success', 'denied', 'error')),
    ADD COLUMN error_code text,
    ADD COLUMN client_ip inet CHECK (pg_catalog.masklen(client_ip) =
        CASE pg_catalog.family(client_ip) WHEN 4 THEN 32 ELSE 128 END),
    ADD COLUMN request_id uuid,
    ADD COLUMN session_hash text CHECK (session_hash ~ '^[0-9a-f]{64}$'),
    ADD COLUMN device_hash text CHECK (device_hash ~ '^[0-9a-f]{64}$'),
    ADD COLUMN metadata jsonb,
    ADD COLUMN before_values jsonb,
    ADD COLUMN after_values jsonb,
    ADD COLUMN chain_index bigint UNIQUE,
    ADD COLUMN chain_version smallint,
    ADD COLUMN prev_hash bytea,
    ADD COLUMN event_hash bytea,
    ADD COLUMN canonical_payload bytea,
    ADD CONSTRAINT identity_audit_chain_fields CHECK (
        (chain_index IS NULL AND chain_version IS NULL AND prev_hash IS NULL
            AND event_hash IS NULL AND canonical_payload IS NULL)
        OR (chain_index IS NOT NULL AND chain_index > 0 AND chain_version IS NOT NULL
            AND chain_version = 1 AND prev_hash IS NOT NULL
            AND pg_catalog.octet_length(prev_hash) = 32 AND event_hash IS NOT NULL
            AND pg_catalog.octet_length(event_hash) = 32 AND canonical_payload IS NOT NULL)
    );

CREATE INDEX identity_audit_client_ip_idx ON public.identity_audit_events (client_ip);
CREATE INDEX identity_audit_action_chain_idx ON public.identity_audit_events (action, chain_index DESC);
CREATE INDEX identity_audit_actor_login_idx ON public.identity_audit_events (actor_login);

CREATE TABLE public.identity_known_devices (
    user_id uuid NOT NULL REFERENCES public.identity_users ON DELETE CASCADE,
    device_hash text NOT NULL CHECK (device_hash ~ '^[0-9a-f]{64}$'),
    first_seen_at timestamptz NOT NULL,
    PRIMARY KEY (user_id, device_hash)
);

CREATE TABLE public.identity_audit_chain (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    stream_id uuid NOT NULL UNIQUE,
    chain_version smallint NOT NULL CHECK (chain_version = 1),
    last_index bigint NOT NULL CHECK (last_index >= 0),
    last_hash bytea NOT NULL CHECK (pg_catalog.octet_length(last_hash) = 32),
    genesis_hash bytea NOT NULL CHECK (pg_catalog.octet_length(genesis_hash) = 32),
    legacy_max_seq bigint NOT NULL CHECK (legacy_max_seq >= 0),
    legacy_count bigint NOT NULL CHECK (legacy_count >= 0),
    created_at timestamptz NOT NULL,
    boundary_payload bytea NOT NULL
);

-- Versioned serialization is an offline-verifier contract. Do not change v1
-- silently with a new database/serialization implementation. Bigints are JSON
-- decimal strings; nullable values are JSON null; timestamptz is fixed UTC/us.
CREATE FUNCTION public.identity_audit_boundary_v1(
    stream_id uuid, legacy_max_seq bigint, legacy_count bigint, created_at timestamptz
) RETURNS bytea
LANGUAGE sql IMMUTABLE PARALLEL SAFE
SET search_path = pg_catalog, pg_temp
AS $function$
    SELECT pg_catalog.convert_to(pg_catalog.jsonb_build_object(
        'streamId', stream_id::text,
        'legacyMaxSeq', legacy_max_seq::text,
        'legacyCount', legacy_count::text,
        'createdAt', pg_catalog.to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8')
$function$;

CREATE FUNCTION public.identity_audit_canonical_v1(
    stream_id uuid, event public.identity_audit_events
) RETURNS bytea
LANGUAGE sql IMMUTABLE PARALLEL SAFE
SET search_path = pg_catalog, pg_temp
AS $function$
    SELECT pg_catalog.convert_to(pg_catalog.jsonb_build_object(
        'streamId', stream_id::text,
        'chainVersion', (event).chain_version,
        'chainIndex', (event).chain_index::text,
        'seq', (event).seq::text,
        'id', (event).id::text,
        'occurredAt', pg_catalog.to_char((event).occurred_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'actorUserId', (event).actor_user_id::text,
        'actorLogin', (event).actor_login,
        'actorDisplayName', (event).actor_display_name,
        'action', (event).action,
        'resourceType', (event).resource_type,
        'resourceId', (event).resource_id::text,
        'companyId', (event).company_id::text,
        'reason', (event).reason,
        'outcome', (event).outcome,
        'errorCode', (event).error_code,
        'clientIp', pg_catalog.host((event).client_ip),
        'details', (event).details,
        'before', (event).before_values,
        'after', (event).after_values,
        'metadata', (event).metadata,
        'requestId', (event).request_id::text,
        'sessionHash', (event).session_hash,
        'deviceHash', (event).device_hash
    )::text, 'UTF8')
$function$;

WITH boundary AS (
    SELECT pg_catalog.gen_random_uuid() AS stream_id,
        COALESCE(pg_catalog.max(seq), 0) AS legacy_max_seq,
        pg_catalog.count(*) AS legacy_count,
        pg_catalog.clock_timestamp() AS created_at
    FROM public.identity_audit_events
), payload AS (
    SELECT boundary.*, public.identity_audit_boundary_v1(
        stream_id, legacy_max_seq, legacy_count, created_at) AS bytes FROM boundary
), genesis AS (
    SELECT payload.*, pg_catalog.sha256(
        pg_catalog.convert_to(E'justixauto-audit-genesis-v1\n', 'UTF8') || bytes) AS hash
    FROM payload
)
INSERT INTO public.identity_audit_chain (
    singleton, stream_id, chain_version, last_index, last_hash, genesis_hash,
    legacy_max_seq, legacy_count, created_at, boundary_payload
)
SELECT true, stream_id, 1, 0, hash, hash, legacy_max_seq, legacy_count, created_at, bytes
FROM genesis;

-- Runtime roles receive only EXECUTE on this narrow lock function from the
-- separately approved privilege template. No role names are guessed here.
CREATE FUNCTION public.identity_audit_lock() RETURNS void
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $function$
BEGIN
    PERFORM 1 FROM public.identity_audit_chain WHERE singleton = true FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'audit chain head is missing' USING ERRCODE = '55000';
    END IF;
END;
$function$;

CREATE FUNCTION public.identity_audit_append() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $function$
DECLARE
    head public.identity_audit_chain%ROWTYPE;
BEGIN
    IF NEW.chain_index IS NOT NULL OR NEW.chain_version IS NOT NULL
        OR NEW.prev_hash IS NOT NULL OR NEW.event_hash IS NOT NULL
        OR NEW.canonical_payload IS NOT NULL THEN
        RAISE EXCEPTION 'audit chain fields are database assigned' USING ERRCODE = '42501';
    END IF;
    SELECT * INTO STRICT head FROM public.identity_audit_chain
        WHERE singleton = true FOR UPDATE;
    NEW.chain_index := head.last_index + 1;
    NEW.chain_version := head.chain_version;
    NEW.prev_hash := head.last_hash;
    NEW.canonical_payload := public.identity_audit_canonical_v1(head.stream_id, NEW);
    NEW.event_hash := pg_catalog.sha256(
        pg_catalog.convert_to(E'justixauto-audit-v1\n', 'UTF8') || NEW.prev_hash || NEW.canonical_payload);
    UPDATE public.identity_audit_chain
        SET last_index = NEW.chain_index, last_hash = NEW.event_hash
        WHERE singleton = true;
    RETURN NEW;
END;
$function$;

-- BEFORE INSERT is also run for rows skipped by ON CONFLICT DO NOTHING. Check
-- every allocated position at commit, including skipped rows, so such inserts
-- cannot commit a head advance without the corresponding immutable event.
CREATE FUNCTION public.identity_audit_head_event_required() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $function$
BEGIN
    IF NEW.last_index <> OLD.last_index + 1
        OR NEW.singleton IS DISTINCT FROM OLD.singleton
        OR NEW.stream_id IS DISTINCT FROM OLD.stream_id
        OR NEW.chain_version IS DISTINCT FROM OLD.chain_version
        OR NEW.genesis_hash IS DISTINCT FROM OLD.genesis_hash
        OR NEW.legacy_max_seq IS DISTINCT FROM OLD.legacy_max_seq
        OR NEW.legacy_count IS DISTINCT FROM OLD.legacy_count
        OR NEW.created_at IS DISTINCT FROM OLD.created_at
        OR NEW.boundary_payload IS DISTINCT FROM OLD.boundary_payload THEN
        RAISE EXCEPTION 'audit boundary is immutable and head must advance once' USING ERRCODE = '23514';
    END IF;
    PERFORM 1 FROM public.identity_audit_events AS event
        WHERE event.chain_index = NEW.last_index AND event.event_hash = NEW.last_hash
            AND event.chain_version = NEW.chain_version AND event.prev_hash = OLD.last_hash
            AND event.canonical_payload = public.identity_audit_canonical_v1(NEW.stream_id, event)
            AND event.event_hash = pg_catalog.sha256(
                pg_catalog.convert_to(E'justixauto-audit-v1\n', 'UTF8') || event.prev_hash || event.canonical_payload);
    IF NOT FOUND THEN
        RAISE EXCEPTION 'audit head has no matching event' USING ERRCODE = '23514';
    END IF;
    RETURN NULL;
END;
$function$;

CREATE FUNCTION public.identity_audit_reject_mutation() RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $function$
BEGIN
    RAISE EXCEPTION 'audit history is append only' USING ERRCODE = '42501';
END;
$function$;

CREATE TRIGGER identity_audit_append BEFORE INSERT ON public.identity_audit_events
    FOR EACH ROW EXECUTE FUNCTION public.identity_audit_append();
CREATE TRIGGER identity_audit_reject_mutation BEFORE UPDATE OR DELETE ON public.identity_audit_events
    FOR EACH ROW EXECUTE FUNCTION public.identity_audit_reject_mutation();
CREATE TRIGGER identity_audit_reject_truncate BEFORE TRUNCATE ON public.identity_audit_events
    FOR EACH STATEMENT EXECUTE FUNCTION public.identity_audit_reject_mutation();
CREATE CONSTRAINT TRIGGER identity_audit_head_event_required
    AFTER UPDATE ON public.identity_audit_chain DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION public.identity_audit_head_event_required();

REVOKE ALL ON TABLE public.identity_audit_chain FROM PUBLIC;
REVOKE UPDATE, DELETE, TRUNCATE, TRIGGER ON TABLE public.identity_audit_events FROM PUBLIC;
REVOKE ALL ON FUNCTION public.identity_audit_lock() FROM PUBLIC;
REVOKE ALL ON FUNCTION public.identity_audit_append() FROM PUBLIC;
REVOKE ALL ON FUNCTION public.identity_audit_head_event_required() FROM PUBLIC;
REVOKE ALL ON FUNCTION public.identity_audit_reject_mutation() FROM PUBLIC;
REVOKE ALL ON FUNCTION public.identity_audit_canonical_v1(uuid, public.identity_audit_events) FROM PUBLIC;
REVOKE ALL ON FUNCTION public.identity_audit_boundary_v1(uuid, bigint, bigint, timestamptz) FROM PUBLIC;

-- Owners/superusers can replace guards and rewrite unanchored history. Runtime
-- owner separation and externally retained checkpoints remain separate gates.
COMMIT;
