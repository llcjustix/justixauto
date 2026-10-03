
-- A bank's or MFO's financing program; published versions are immutable so
-- old calculations never change.
CREATE TABLE financing_programs (
    id                  uuid        PRIMARY KEY,
    provider_company_id uuid        NOT NULL,
    status              text        NOT NULL CHECK (status IN ('draft', 'published', 'withdrawn')),
    published_version   integer,
    status_reason       text        NOT NULL DEFAULT '',
    version             bigint      NOT NULL CHECK (version >= 1),
    created_at          timestamptz NOT NULL,
    updated_at          timestamptz NOT NULL
);
CREATE INDEX programs_provider_idx ON financing_programs (provider_company_id);

CREATE TABLE financing_program_versions (
    program_id     uuid        NOT NULL REFERENCES financing_programs,
    number         integer     NOT NULL CHECK (number >= 1),
    name           text        NOT NULL,
    currency       text        NOT NULL,
    terms          jsonb       NOT NULL,
    eligibility    jsonb       NOT NULL,
    policy_id      text        NOT NULL,
    policy_version integer     NOT NULL,
    created_by     uuid        NOT NULL,
    created_at     timestamptz NOT NULL,
    PRIMARY KEY (program_id, number)
);

-- One shared application: the seller's and the provider's views of it.
CREATE TABLE financing_applications (
    id                    uuid        PRIMARY KEY,
    seller_company_id     uuid        NOT NULL,
    provider_company_id   uuid        NOT NULL,
    deal_id               uuid        NOT NULL,
    program_id            uuid        REFERENCES financing_programs,
    program_version       integer,
    calculation           jsonb,
    calculation_digest    text        NOT NULL DEFAULT '',
    status                text        NOT NULL CHECK (status IN ('draft', 'submitted', 'review', 'needs-info', 'terms', 'agreed', 'declined')),
    snapshot              jsonb,
    current_terms_version integer,
    version               bigint      NOT NULL CHECK (version >= 1),
    created_by            uuid        NOT NULL,
    created_at            timestamptz NOT NULL,
    updated_at            timestamptz NOT NULL,
    submitted_at          timestamptz
);
-- One open application per sale; after a decline the seller may apply again.
CREATE UNIQUE INDEX applications_open_deal_key ON financing_applications (deal_id) WHERE status <> 'declined';
CREATE INDEX applications_provider_idx ON financing_applications (provider_company_id, status) WHERE status <> 'draft';

-- The provider's proposed terms: numbered, immutable, recalculated by the server.
CREATE TABLE financing_terms_versions (
    application_id uuid        NOT NULL REFERENCES financing_applications,
    number         integer     NOT NULL CHECK (number >= 1),
    calculation    jsonb       NOT NULL,
    note           text        NOT NULL,
    created_by     uuid        NOT NULL,
    created_at     timestamptz NOT NULL,
    PRIMARY KEY (application_id, number)
);

CREATE TABLE financing_messages (
    id             uuid        PRIMARY KEY,
    seq            bigint      GENERATED ALWAYS AS IDENTITY UNIQUE,
    application_id uuid        NOT NULL REFERENCES financing_applications,
    kind           text        NOT NULL,
    request_id     uuid,
    note           text        NOT NULL DEFAULT '',
    terms_version  integer,
    company_id     uuid        NOT NULL,
    actor_user_id  uuid        NOT NULL,
    created_at     timestamptz NOT NULL
);
CREATE INDEX financing_messages_idx ON financing_messages (application_id, seq);
