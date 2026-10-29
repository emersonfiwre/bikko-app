-- Runtime no longer mutates the schema. This migration reconciles older local volumes.
ALTER TABLE users ADD COLUMN IF NOT EXISTS location_name TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS location_geom GEOGRAPHY(Point,4326);
UPDATE users SET phone = NULLIF(regexp_replace(phone, '[^0-9]', '', 'g'), '') WHERE phone IS NOT NULL;
UPDATE users SET phone = substring(phone FROM 3) WHERE length(phone)=13 AND left(phone,2)='55';
-- Deliberately fail on duplicate identities; never merge accounts by suffix.
CREATE UNIQUE INDEX IF NOT EXISTS users_active_phone_unique ON users(phone) WHERE deleted_at IS NULL AND phone IS NOT NULL;
ALTER TABLE services ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS contact_name VARCHAR(255) DEFAULT '';
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS contact_phone VARCHAR(50) DEFAULT '';
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS account_phone VARCHAR(50) DEFAULT '';
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS service_title VARCHAR(255) DEFAULT '';
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS category_name VARCHAR(255) DEFAULT '';
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS preferred_date VARCHAR(100) DEFAULT '';
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS client_photos JSONB DEFAULT '[]';
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS client_evaluated BOOLEAN DEFAULT FALSE;
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS bikker_evaluated BOOLEAN DEFAULT FALSE;
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS client_rating_given INTEGER DEFAULT 0;
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS technical_report TEXT DEFAULT '';
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS request_key VARCHAR(128);
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS request_fingerprint TEXT;
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS cancellation_reason TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS quote_request_idempotency_unique ON quote_requests(customer_id,request_key) WHERE request_key IS NOT NULL;
ALTER TABLE quote_requests ALTER COLUMN service_id DROP NOT NULL;
ALTER TABLE orders ALTER COLUMN service_id DROP NOT NULL;
ALTER TABLE reviews ALTER COLUMN service_id DROP NOT NULL;
ALTER TABLE reviews DROP CONSTRAINT IF EXISTS reviews_order_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_reviews_order_reviewer ON reviews(order_id,reviewer_id);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS accepted_negotiation_id UUID REFERENCES quote_negotiations(id);
ALTER TABLE quote_negotiations ADD COLUMN IF NOT EXISTS request_key VARCHAR(128);
CREATE UNIQUE INDEX IF NOT EXISTS negotiation_request_key_unique ON quote_negotiations(quote_request_id,sender_id,request_key) WHERE request_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS quote_requests_customer_created ON quote_requests(customer_id,created_at DESC);
CREATE INDEX IF NOT EXISTS quote_requests_provider_created ON quote_requests(bikker_id,created_at DESC);
CREATE INDEX IF NOT EXISTS negotiations_quote_created ON quote_negotiations(quote_request_id,created_at,id);
CREATE TABLE IF NOT EXISTS user_devices (
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 app_type TEXT NOT NULL CHECK(app_type IN ('client','provider')),
 token TEXT NOT NULL UNIQUE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY(user_id,app_type,token)
);
CREATE TABLE IF NOT EXISTS notification_outbox (
 id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
 user_id UUID NOT NULL REFERENCES users(id),
 title TEXT NOT NULL,
 body TEXT NOT NULL,
 data JSONB NOT NULL DEFAULT '{}',
 attempts INTEGER NOT NULL DEFAULT 0,
 available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 delivered_at TIMESTAMPTZ,
 last_error TEXT,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS outbox_pending ON notification_outbox(available_at,created_at) WHERE delivered_at IS NULL;
