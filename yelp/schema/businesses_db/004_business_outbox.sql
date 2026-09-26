CREATE TABLE business_outbox (
    id              BIGSERIAL PRIMARY KEY,
    business_id     TEXT NOT NULL,    -- match businesses.id's type
    operation       TEXT NOT NULL DEFAULT 'upsert',  -- 'upsert' | 'delete'
    processed       BOOLEAN NOT NULL DEFAULT false,
    attempts        INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_business_outbox_unprocessed
    ON business_outbox (id) WHERE processed = false;


-- business_outbox_fn: trigger function to write to the outbox table on every insert/update/delete on businesses table
-- 1. The function that runs on every insert/update/delete
CREATE OR REPLACE FUNCTION business_outbox_fn() RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN
    INSERT INTO business_outbox (business_id, operation)
    VALUES (OLD.id, 'delete');
    RETURN OLD;
  END IF;

  IF TG_OP = 'UPDATE' AND NEW IS NOT DISTINCT FROM OLD THEN
    RETURN NEW; -- nothing actually changed, skip writing an outbox row
  END IF;

  INSERT INTO business_outbox (business_id, operation)
  VALUES (NEW.id, 'upsert');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 2. The trigger that attaches the function to the businesses table
CREATE TRIGGER business_outbox_trg
AFTER INSERT OR UPDATE OR DELETE ON businesses
FOR EACH ROW EXECUTE FUNCTION business_outbox_fn();