-- 000006_create_client_reviews.up.sql
CREATE TABLE IF NOT EXISTS client_reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID REFERENCES orders(id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bikker_id UUID NOT NULL REFERENCES bikkers(id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    technical_report TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_client_reviews_client_id ON client_reviews(client_id);
CREATE INDEX IF NOT EXISTS idx_client_reviews_bikker_id ON client_reviews(bikker_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_client_reviews_order_bikker ON client_reviews(order_id, bikker_id);
