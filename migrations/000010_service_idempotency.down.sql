DROP INDEX IF EXISTS services_bikker_request_key_unique;
ALTER TABLE services DROP COLUMN IF EXISTS request_fingerprint;
ALTER TABLE services DROP COLUMN IF EXISTS request_key;
