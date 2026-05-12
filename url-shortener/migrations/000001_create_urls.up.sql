CREATE TABLE IF NOT EXISTS urls (
    id          BIGSERIAL       PRIMARY KEY,
    code        VARCHAR(12)     NOT NULL UNIQUE,   -- Base62 encoded id, e.g. "4c92"
    original    TEXT            NOT NULL,           -- the full original URL
    alias       VARCHAR(64)     UNIQUE,             -- optional custom alias, e.g. "my-link"
    expires_at  TIMESTAMPTZ,                        -- NULL = never expires
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- Fast lookup by code (redirect path — hit on every request)
CREATE INDEX IF NOT EXISTS idx_urls_code ON urls (code);

-- Fast lookup by alias (used when a custom alias is provided)
CREATE INDEX IF NOT EXISTS idx_urls_alias ON urls (alias) WHERE alias IS NOT NULL;

-- Used by a cleanup job to find and purge expired links
CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls (expires_at) WHERE expires_at IS NOT NULL;