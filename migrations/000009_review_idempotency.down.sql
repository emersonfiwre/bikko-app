DROP INDEX IF EXISTS client_reviews_bikker_request_key_unique;
ALTER TABLE client_reviews DROP COLUMN IF EXISTS request_fingerprint;
ALTER TABLE client_reviews DROP COLUMN IF EXISTS request_key;

DROP INDEX IF EXISTS reviews_reviewer_request_key_unique;
ALTER TABLE reviews DROP COLUMN IF EXISTS request_fingerprint;
ALTER TABLE reviews DROP COLUMN IF EXISTS request_key;
