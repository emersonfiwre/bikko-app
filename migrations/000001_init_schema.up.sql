-- Enable Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS postgis;

-- Enum Types
CREATE TYPE quote_status AS ENUM ('PENDING', 'NEGOTIATING', 'ACCEPTED', 'REJECTED', 'EXPIRED');
CREATE TYPE order_status AS ENUM ('SCHEDULED', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED');

-- 1. USERS Table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    full_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(50),
    cpf VARCHAR(14) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    profile_photo_url VARCHAR(512),
    rating DECIMAL(3, 2) DEFAULT 0.00,
    total_ratings INTEGER DEFAULT 0,
    is_bikker BOOLEAN DEFAULT FALSE,
    notifications_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. BIKKERS Table (1:1 with users)
CREATE TABLE IF NOT EXISTS bikkers (
    id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    profession VARCHAR(100) NOT NULL,
    experience_years VARCHAR(50),
    description TEXT,
    location_name VARCHAR(255),
    location_geom GEOGRAPHY(Point, 4326),
    rating DECIMAL(3, 2) DEFAULT 0.00,
    total_reviews INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. CATEGORIES Table
CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    icon_url VARCHAR(512),
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. SERVICES Table
CREATE TABLE IF NOT EXISTS services (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bikker_id UUID NOT NULL REFERENCES bikkers(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    thumbnail_url VARCHAR(512),
    location_geom GEOGRAPHY(Point, 4326),
    reviews_average DECIMAL(3, 2) DEFAULT 0.00,
    total_reviews INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 5. SERVICE_PHOTOS Table
CREATE TABLE IF NOT EXISTS service_photos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    photo_url VARCHAR(512) NOT NULL,
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 6. QUOTE_REQUESTS Table
CREATE TABLE IF NOT EXISTS quote_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bikker_id UUID NOT NULL REFERENCES bikkers(id) ON DELETE CASCADE,
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    status quote_status DEFAULT 'PENDING',
    initial_description TEXT NOT NULL,
    desired_date TIMESTAMP WITH TIME ZONE,
    location_address VARCHAR(255),
    location_geom GEOGRAPHY(Point, 4326),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 7. QUOTE_NEGOTIATIONS Table
CREATE TABLE IF NOT EXISTS quote_negotiations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    quote_request_id UUID NOT NULL REFERENCES quote_requests(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    proposed_price DECIMAL(10, 2),
    photos JSONB DEFAULT '[]'::jsonb,
    is_counter_offer BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 8. ORDERS Table (1:1 with QUOTE_REQUESTS)
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    quote_request_id UUID NOT NULL UNIQUE REFERENCES quote_requests(id) ON DELETE RESTRICT,
    customer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bikker_id UUID NOT NULL REFERENCES bikkers(id) ON DELETE CASCADE,
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    status order_status DEFAULT 'SCHEDULED',
    final_price DECIMAL(10, 2) NOT NULL,
    scheduled_date TIMESTAMP WITH TIME ZONE NOT NULL,
    cancellation_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 9. REVIEWS Table (1:1 with completed ORDERS)
CREATE TABLE IF NOT EXISTS reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    reviewer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bikker_id UUID NOT NULL REFERENCES bikkers(id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    like_count INTEGER DEFAULT 0,
    dislike_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Spatial GiST Indexes for Fast Proximity Queries
CREATE INDEX IF NOT EXISTS idx_bikkers_location_gist ON bikkers USING GIST (location_geom);
CREATE INDEX IF NOT EXISTS idx_services_location_gist ON services USING GIST (location_geom);
CREATE INDEX IF NOT EXISTS idx_quote_requests_location_gist ON quote_requests USING GIST (location_geom);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_services_bikker_id ON services(bikker_id);
CREATE INDEX IF NOT EXISTS idx_services_category_id ON services(category_id);
CREATE INDEX IF NOT EXISTS idx_quote_requests_customer_id ON quote_requests(customer_id);
CREATE INDEX IF NOT EXISTS idx_quote_requests_bikker_id ON quote_requests(bikker_id);
CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_bikker_id ON orders(bikker_id);
