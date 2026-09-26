CREATE TABLE business_categories (
    business_id     TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    category_id     INT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (business_id, category_id)
);

-- CREATE INDEX idx_business_categories_category ON business_categories (category_id);