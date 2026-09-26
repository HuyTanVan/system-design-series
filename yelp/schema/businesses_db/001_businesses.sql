CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE businesses (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    address         TEXT,
    city            TEXT,
    state           TEXT,
    postal_code     TEXT,
    location        GEOGRAPHY(POINT, 4326) NOT NULL,
    avg_rating      NUMERIC(2,1) DEFAULT 0,
    review_count    INT DEFAULT 0,
    is_open         BOOLEAN DEFAULT true,
    attributes      JSONB,
    hours           JSONB,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

-- CREATE INDEX idx_businesses_location ON businesses USING GIST (location);


