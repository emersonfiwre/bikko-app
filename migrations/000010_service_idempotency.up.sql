ALTER TABLE services ADD COLUMN IF NOT EXISTS request_key VARCHAR(128);
ALTER TABLE services ADD COLUMN IF NOT EXISTS request_fingerprint TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS services_bikker_request_key_unique
    ON services(bikker_id, request_key) WHERE request_key IS NOT NULL;
