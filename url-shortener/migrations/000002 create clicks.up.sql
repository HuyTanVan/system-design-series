CREATE TABLE IF NOT EXISTS clicks (
    id          BIGSERIAL       PRIMARY KEY,
    url_id      BIGINT          NOT NULL REFERENCES urls (id) ON DELETE CASCADE,
    ip          INET,                               -- visitor IP (nullable for privacy)
    user_agent  TEXT,                               -- raw User-Agent header
    referer     TEXT,                               -- HTTP Referer header
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- Core analytics query: "give me all clicks for url X"
CREATE INDEX IF NOT EXISTS idx_clicks_url_id ON clicks (url_id);

-- Time-range analytics: "clicks for url X in the last 7 days"
CREATE INDEX IF NOT EXISTS idx_clicks_url_id_created_at ON clicks (url_id, created_at DESC);