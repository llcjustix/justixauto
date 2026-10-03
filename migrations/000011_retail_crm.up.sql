
-- Company-local natural persons (personal data stays with the company).
CREATE TABLE retail_customers (
    id           uuid        PRIMARY KEY,
    company_id   uuid        NOT NULL,
    display_name text        NOT NULL CHECK (length(display_name) BETWEEN 1 AND 200),
    phone        text        NOT NULL DEFAULT '' CHECK (length(phone) <= 50),
    version      bigint      NOT NULL CHECK (version >= 1),
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL
);
CREATE INDEX customers_company_idx ON retail_customers (company_id, display_name);

CREATE TABLE retail_leads (
    id               uuid        PRIMARY KEY,
    company_id       uuid        NOT NULL,
    branch_id        uuid        NOT NULL,
    customer_id      uuid        NOT NULL REFERENCES retail_customers,
    source           text        NOT NULL CHECK (source IN ('website', 'telegram', 'phone', 'manual')),
    stage            text        NOT NULL CHECK (stage IN ('new', 'contacted', 'qualified', 'test-drive', 'negotiation', 'won', 'lost')),
    assigned_user_id uuid,
    lost_reason      text        NOT NULL DEFAULT '',
    deal_id          uuid,
    version          bigint      NOT NULL CHECK (version >= 1),
    created_at       timestamptz NOT NULL,
    updated_at       timestamptz NOT NULL
);
CREATE INDEX leads_company_idx ON retail_leads (company_id, branch_id, stage);

-- Contact history is append-only.
CREATE TABLE retail_lead_contacts (
    id          uuid        PRIMARY KEY,
    lead_id     uuid        NOT NULL REFERENCES retail_leads,
    channel     text        NOT NULL CHECK (channel IN ('phone', 'telegram', 'visit', 'email', 'other')),
    note        text        NOT NULL CHECK (length(note) BETWEEN 1 AND 2000),
    actor_id    uuid        NOT NULL,
    occurred_at timestamptz NOT NULL
);
CREATE INDEX lead_contacts_idx ON retail_lead_contacts (lead_id, occurred_at);

CREATE TABLE retail_tasks (
    id            uuid        PRIMARY KEY,
    company_id    uuid        NOT NULL,
    customer_id   uuid        NOT NULL REFERENCES retail_customers,
    lead_id       uuid        REFERENCES retail_leads,
    deal_id       uuid,
    owner_user_id uuid        NOT NULL,
    due_at        timestamptz NOT NULL,
    title         text        NOT NULL CHECK (length(title) BETWEEN 1 AND 300),
    status        text        NOT NULL CHECK (status IN ('open', 'completed')),
    completed_at  timestamptz,
    completed_by  uuid,
    version       bigint      NOT NULL CHECK (version >= 1),
    created_at    timestamptz NOT NULL
);
CREATE INDEX tasks_owner_idx ON retail_tasks (company_id, owner_user_id, status, due_at);

-- A retail offer for an existing vehicle; it never rewrites vehicle facts.
CREATE TABLE retail_listings (
    id                 uuid        PRIMARY KEY,
    company_id         uuid        NOT NULL,
    vehicle_id         uuid        NOT NULL,
    text               text        NOT NULL CHECK (length(text) <= 5000),
    asking_price_minor text        NOT NULL,
    currency           text        NOT NULL,
    status             text        NOT NULL CHECK (status IN ('draft', 'published', 'withdrawn')),
    version            bigint      NOT NULL CHECK (version >= 1),
    created_at         timestamptz NOT NULL,
    updated_at         timestamptz NOT NULL
);
CREATE UNIQUE INDEX listings_open_vehicle_key ON retail_listings (vehicle_id) WHERE status <> 'withdrawn';

CREATE TABLE retail_events (
    id            uuid        PRIMARY KEY,
    seq           bigint      GENERATED ALWAYS AS IDENTITY UNIQUE,
    company_id    uuid        NOT NULL,
    event_type    text        NOT NULL,
    resource_type text        NOT NULL,
    resource_id   uuid        NOT NULL,
    actor_user_id uuid        NOT NULL,
    occurred_at   timestamptz NOT NULL,
    reason        text        NOT NULL DEFAULT '',
    details       jsonb       NOT NULL DEFAULT '{}'
);
CREATE INDEX retail_events_resource_idx ON retail_events (resource_type, resource_id, seq);
