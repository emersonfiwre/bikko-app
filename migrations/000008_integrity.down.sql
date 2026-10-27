DROP TABLE IF EXISTS notification_outbox CASCADE;
DROP TABLE IF EXISTS user_devices CASCADE;

DROP INDEX IF EXISTS outbox_pending;
DROP INDEX IF EXISTS negotiations_quote_created;
DROP INDEX IF EXISTS quote_requests_provider_created;
DROP INDEX IF EXISTS quote_requests_customer_created;
DROP INDEX IF EXISTS negotiation_request_key_unique;
DROP INDEX IF EXISTS idx_reviews_order_reviewer;
DROP INDEX IF EXISTS quote_request_idempotency_unique;
DROP INDEX IF EXISTS users_active_phone_unique;

ALTER TABLE orders DROP COLUMN IF EXISTS accepted_negotiation_id;
ALTER TABLE quote_negotiations DROP COLUMN IF EXISTS request_key;

ALTER TABLE quote_requests DROP COLUMN IF EXISTS cancellation_reason;
ALTER TABLE quote_requests DROP COLUMN IF EXISTS request_fingerprint;
ALTER TABLE quote_requests DROP COLUMN IF EXISTS request_key;
ALTER TABLE quote_requests DROP COLUMN IF EXISTS technical_report;
ALTER TABLE quote_requests DROP COLUMN IF EXISTS client_rating_given;
ALTER TABLE quote_requests DROP COLUMN IF EXISTS bikker_evaluated;
ALTER TABLE quote_requests DROP COLUMN IF EXISTS client_evaluated;
ALTER TABLE quote_requests DROP COLUMN IF EXISTS client_photos;
ALTER TABLE quote_requests DROP COLUMN IF EXISTS preferred_date;
ALTER TABLE quote_requests DROP COLUMN IF EXISTS category_name;
ALTER TABLE quote_requests DROP COLUMN IF EXISTS service_title;
ALTER TABLE quote_requests DROP COLUMN IF EXISTS account_phone;
ALTER TABLE quote_requests DROP COLUMN IF EXISTS contact_phone;
ALTER TABLE quote_requests DROP COLUMN IF EXISTS contact_name;

ALTER TABLE services DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE users DROP COLUMN IF EXISTS location_geom;
ALTER TABLE users DROP COLUMN IF EXISTS location_name;
