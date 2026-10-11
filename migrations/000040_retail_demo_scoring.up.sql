CREATE TABLE retail_demo_scoring_runs (
    id uuid PRIMARY KEY,
    company_id uuid NOT NULL,
    deal_id uuid NOT NULL REFERENCES retail_deals(id) ON DELETE RESTRICT,
    deal_revision bigint NOT NULL CHECK (deal_revision >= 0),
    terms_digest char(64) NOT NULL,
    mode text NOT NULL CHECK (mode = 'demo'),
    result text NOT NULL CHECK (result = 'completed'),
    completed_by uuid NOT NULL,
    completed_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL
);
CREATE INDEX retail_demo_scoring_runs_current_idx ON retail_demo_scoring_runs(company_id, deal_id, completed_at DESC, id DESC);
