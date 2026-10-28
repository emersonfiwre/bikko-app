ALTER TABLE reviews ADD COLUMN IF NOT EXISTS request_key VARCHAR(128);
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS request_fingerprint TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS reviews_reviewer_request_key_unique
    ON reviews(reviewer_id, request_key) WHERE request_key IS NOT NULL;

ALTER TABLE client_reviews ADD COLUMN IF NOT EXISTS request_key VARCHAR(128);
ALTER TABLE client_reviews ADD COLUMN IF NOT EXISTS request_fingerprint TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS client_reviews_bikker_request_key_unique
    ON client_reviews(bikker_id, request_key) WHERE request_key IS NOT NULL;
