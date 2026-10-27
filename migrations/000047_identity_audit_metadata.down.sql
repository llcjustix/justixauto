BEGIN;

-- Prevent a concurrent append between the evidence check and schema removal.
LOCK TABLE public.identity_audit_events IN ACCESS EXCLUSIVE MODE;
LOCK TABLE public.identity_audit_chain IN ACCESS EXCLUSIVE MODE;
DO $block$
BEGIN
    IF EXISTS (SELECT 1 FROM public.identity_audit_events WHERE chain_index IS NOT NULL)
        OR EXISTS (SELECT 1 FROM public.identity_audit_chain WHERE last_index <> 0) THEN
        RAISE EXCEPTION 'cannot roll back populated audit evidence' USING ERRCODE = '55000';
    END IF;
END;
$block$;

DROP TRIGGER identity_audit_append ON public.identity_audit_events;
DROP TRIGGER identity_audit_reject_mutation ON public.identity_audit_events;
DROP TRIGGER identity_audit_reject_truncate ON public.identity_audit_events;
DROP TRIGGER identity_audit_head_event_required ON public.identity_audit_chain;
DROP FUNCTION public.identity_audit_append();
DROP FUNCTION public.identity_audit_head_event_required();
DROP FUNCTION public.identity_audit_reject_mutation();
DROP FUNCTION public.identity_audit_lock();
DROP FUNCTION public.identity_audit_canonical_v1(uuid, public.identity_audit_events);
DROP FUNCTION public.identity_audit_boundary_v1(uuid, bigint, bigint, timestamptz);
DROP TABLE public.identity_audit_chain;
DROP TABLE public.identity_known_devices;
DROP INDEX public.identity_audit_client_ip_idx;
DROP INDEX public.identity_audit_action_chain_idx;
DROP INDEX public.identity_audit_actor_login_idx;
ALTER TABLE public.identity_audit_events
    DROP CONSTRAINT identity_audit_chain_fields,
    DROP COLUMN actor_login,
    DROP COLUMN actor_display_name,
    DROP COLUMN outcome,
    DROP COLUMN error_code,
    DROP COLUMN client_ip,
    DROP COLUMN request_id,
    DROP COLUMN session_hash,
    DROP COLUMN device_hash,
    DROP COLUMN metadata,
    DROP COLUMN before_values,
    DROP COLUMN after_values,
    DROP COLUMN chain_index,
    DROP COLUMN chain_version,
    DROP COLUMN prev_hash,
    DROP COLUMN event_hash,
    DROP COLUMN canonical_payload,
    ALTER COLUMN resource_id SET NOT NULL;

COMMIT;
