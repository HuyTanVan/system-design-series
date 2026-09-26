CREATE TABLE reviews (
    id              TEXT PRIMARY KEY,
    business_id     TEXT NOT NULL,
    user_id         TEXT NOT NULL,
    rating          SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    text            TEXT,
    useful          INT DEFAULT 0,
    funny           INT DEFAULT 0,
    cool            INT DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL
);

-- no indexing=29s, with indexing=76.5, b_id=XQfwVwDr-v0ZS3_CbbE5Xw
-- CREATE INDEX idx_reviews_business_id ON reviews (business_id);
