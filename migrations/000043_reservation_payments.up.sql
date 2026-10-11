CREATE TABLE commerce_reservation_evidence (
    id uuid PRIMARY KEY,
    reservation_id uuid NOT NULL REFERENCES commerce_reservations(id),
    terms_digest text NOT NULL,
    terms_revision bigint NOT NULL CHECK (terms_revision >= 1),
    amount_minor text NOT NULL CHECK (amount_minor ~ '^[1-9][0-9]{0,37}$'),
    currency text NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    paid_on date NOT NULL,
    external_reference text NOT NULL,
    attachment_ids jsonb NOT NULL DEFAULT '[]'::jsonb,
    status text NOT NULL CHECK (status IN ('submitted','accepted','rejected')),
    decision_reason text NOT NULL DEFAULT '',
    submitted_by uuid NOT NULL,
    decided_by uuid,
    version bigint NOT NULL CHECK (version >= 1),
    created_at timestamptz NOT NULL,
    decided_at timestamptz
);
CREATE INDEX reservation_evidence_source_idx ON commerce_reservation_evidence(reservation_id, created_at, id);

CREATE TABLE commerce_reservation_refunds (
    reservation_id uuid PRIMARY KEY REFERENCES commerce_reservations(id),
    status text NOT NULL CHECK (status IN ('requested','confirmed')),
    requested_by uuid NOT NULL,
    requested_at timestamptz NOT NULL,
    request_reason text NOT NULL,
    amount_minor text NOT NULL DEFAULT '',
    currency text NOT NULL DEFAULT '',
    external_reference text NOT NULL DEFAULT '',
    attachment_ids jsonb NOT NULL DEFAULT '[]'::jsonb,
    refunded_on date,
    confirmed_by uuid,
    confirmed_at timestamptz,
    CHECK (status = 'requested' OR
           (amount_minor ~ '^[1-9][0-9]{0,37}$' AND currency ~ '^[A-Z]{3}$'
            AND external_reference <> '' AND refunded_on IS NOT NULL
            AND confirmed_by IS NOT NULL AND confirmed_at IS NOT NULL))
);

CREATE TABLE commerce_reservation_credits (
    reservation_id uuid PRIMARY KEY REFERENCES commerce_reservations(id),
    order_id uuid NOT NULL UNIQUE REFERENCES commerce_orders(id),
    buyer_company_id uuid NOT NULL,
    supplier_company_id uuid NOT NULL,
    amount_minor text NOT NULL CHECK (amount_minor ~ '^[1-9][0-9]{0,37}$'),
    currency text NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    evidence_ids jsonb NOT NULL,
    invoice_id uuid UNIQUE REFERENCES commerce_invoices(id),
    created_at timestamptz NOT NULL,
    invoice_attached_at timestamptz,
    CHECK ((invoice_id IS NULL) = (invoice_attached_at IS NULL))
);
