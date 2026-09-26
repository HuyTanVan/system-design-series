CREATE TABLE review_outbox (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id     TEXT NOT NULL,
    rating          SMALLINT NOT NULL,
    processed       BOOLEAN DEFAULT false,
    created_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_outbox_unprocessed ON review_outbox (processed) WHERE processed = false;