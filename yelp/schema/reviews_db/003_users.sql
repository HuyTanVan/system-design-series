CREATE TABLE users (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    -- email           TEXT UNIQUE,       -- null for imported Yelp users, required on signup
    -- password_hash   TEXT,              -- null for imported Yelp users, set on signup
    review_count    INT DEFAULT 0,
    average_stars   NUMERIC(3,2),
    created_at      TIMESTAMPTZ DEFAULT now()
);