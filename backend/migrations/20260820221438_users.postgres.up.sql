CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE counties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    region VARCHAR(100) NOT NULL,
    latitude NUMERIC(9, 6),
    longitude NUMERIC(9, 6),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
    CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180)
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    password_hash TEXT NOT NULL,
    first_name VARCHAR(100) NOT NULL DEFAULT '',
    last_name VARCHAR(100) NOT NULL DEFAULT '',
    role VARCHAR(20) NOT NULL,
    business_name VARCHAR(200) NOT NULL DEFAULT '',
    phone VARCHAR(30),
    location VARCHAR(255) NOT NULL DEFAULT '',
    county_id UUID REFERENCES counties(id) ON DELETE SET NULL,
    lightning_address VARCHAR(255),
    balance_sats BIGINT NOT NULL DEFAULT 0 CHECK (balance_sats >= 0),
    rating NUMERIC(3, 2) NOT NULL DEFAULT 0 CHECK (rating BETWEEN 0 AND 5),
    completed_pickups INTEGER NOT NULL DEFAULT 0 CHECK (completed_pickups >= 0),
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (role IN ('producer', 'farmer', 'admin'))
);

CREATE UNIQUE INDEX users_email_unique_idx ON users (LOWER(email));
CREATE UNIQUE INDEX users_phone_unique_idx ON users (phone) WHERE phone IS NOT NULL AND phone <> '';
CREATE UNIQUE INDEX users_lightning_address_unique_idx ON users (LOWER(lightning_address)) WHERE lightning_address IS NOT NULL AND lightning_address <> '';
CREATE INDEX users_county_id_idx ON users (county_id);
CREATE INDEX users_role_idx ON users (role);

CREATE TABLE listings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    producer_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title VARCHAR(255) NOT NULL,
    producer_type VARCHAR(50) NOT NULL CHECK (producer_type IN ('Hotel', 'Abattoir', 'Food Market', 'Agro-Processor', 'Brewery', 'Grain Mill')),
    category VARCHAR(50) NOT NULL CHECK (category IN ('Spent Grain', 'Vegetable Scraps', 'Abattoir Offal', 'Poultry Litter', 'Fish Offal', 'Fruit Scraps', 'Milling Residue')),
    intended_uses TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    quantity_kg NUMERIC(12, 2) NOT NULL CHECK (quantity_kg > 0),
    available_quantity_kg NUMERIC(12, 2) NOT NULL CHECK (available_quantity_kg >= 0 AND available_quantity_kg <= quantity_kg),
    price_sats BIGINT NOT NULL CHECK (price_sats >= 0),
    price_sats_per_kg NUMERIC(14, 4) NOT NULL CHECK (price_sats_per_kg >= 0),
    location_name VARCHAR(255) NOT NULL,
    county_id UUID REFERENCES counties(id) ON DELETE SET NULL,
    latitude NUMERIC(9, 6) CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
    longitude NUMERIC(9, 6) CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
    map_x NUMERIC(6, 2) CHECK (map_x IS NULL OR map_x BETWEEN 0 AND 100),
    map_y NUMERIC(6, 2) CHECK (map_y IS NULL OR map_y BETWEEN 0 AND 100),
    is_processed BOOLEAN NOT NULL DEFAULT FALSE,
    processing_details TEXT NOT NULL DEFAULT '',
    moisture_content NUMERIC(5, 2) CHECK (moisture_content IS NULL OR moisture_content BETWEEN 0 AND 100),
    availability VARCHAR(30) NOT NULL CHECK (availability IN ('Daily Recurring', 'Weekly Recurring', 'One-off Batch')),
    pickup_window VARCHAR(255) NOT NULL,
    image_url TEXT,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'reserved', 'sold', 'inactive')),
    posted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX listings_producer_id_idx ON listings (producer_id);
CREATE INDEX listings_county_id_idx ON listings (county_id);
CREATE INDEX listings_category_status_idx ON listings (category, status);
CREATE INDEX listings_posted_at_idx ON listings (posted_at DESC);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES listings(id) ON DELETE RESTRICT,
    producer_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    farmer_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    quantity_ordered_kg NUMERIC(12, 2) NOT NULL CHECK (quantity_ordered_kg > 0),
    total_sats BIGINT NOT NULL CHECK (total_sats >= 0),
    total_kes NUMERIC(14, 2) NOT NULL CHECK (total_kes >= 0),
    status VARCHAR(30) NOT NULL DEFAULT 'Pending Payment' CHECK (status IN ('Pending Payment', 'Paid & Scheduled', 'In Transit', 'Completed', 'Cancelled')),
    pickup_date DATE,
    pickup_time_slot VARCHAR(100),
    qr_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ,
    CHECK (producer_id <> farmer_id)
);

CREATE INDEX orders_listing_id_idx ON orders (listing_id);
CREATE INDEX orders_producer_id_idx ON orders (producer_id);
CREATE INDEX orders_farmer_id_idx ON orders (farmer_id);
CREATE INDEX orders_status_created_at_idx ON orders (status, created_at DESC);

CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID REFERENCES orders(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    payment_request TEXT NOT NULL,
    payment_hash VARCHAR(255) UNIQUE,
    payment_secret TEXT,
    payment_preimage TEXT,
    amount_sats BIGINT NOT NULL CHECK (amount_sats > 0),
    amount_kes NUMERIC(14, 2) CHECK (amount_kes IS NULL OR amount_kes >= 0),
    memo TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'expired', 'cancelled', 'failed')),
    expires_at TIMESTAMPTZ,
    settled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX invoices_order_id_idx ON invoices (order_id);
CREATE INDEX invoices_user_id_idx ON invoices (user_id);
CREATE INDEX invoices_status_expires_at_idx ON invoices (status, expires_at);

CREATE TABLE wallet_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    invoice_id UUID REFERENCES invoices(id) ON DELETE SET NULL,
    transaction_type VARCHAR(30) NOT NULL CHECK (transaction_type IN ('deposit', 'withdrawal', 'order_payment', 'order_receipt', 'refund')),
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('INBOUND', 'OUTBOUND')),
    settlement_method VARCHAR(20) NOT NULL CHECK (settlement_method IN ('LIGHTNING', 'MPESA_C2B', 'MPESA_B2C')),
    status VARCHAR(30) NOT NULL CHECK (status IN ('RECEIVED', 'CONVERTING', 'PAYOUT_PENDING', 'SETTLED', 'EXPIRED', 'FAILED', 'TIMED_OUT')),
    amount_sats BIGINT NOT NULL DEFAULT 0 CHECK (amount_sats >= 0),
    fiat_amount_kes NUMERIC(14, 2) NOT NULL DEFAULT 0 CHECK (fiat_amount_kes >= 0),
    exchange_rate NUMERIC(18, 8),
    exchange_rate_source VARCHAR(100),
    exchange_rate_timestamp TIMESTAMPTZ,
    mpesa_transaction_id VARCHAR(100),
    mpesa_type VARCHAR(10),
    payer_msisdn VARCHAR(30),
    payee_msisdn VARCHAR(30),
    payment_hash VARCHAR(255),
    provider_reference VARCHAR(255),
    memo TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ,
    settled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX wallet_transactions_mpesa_id_unique_idx ON wallet_transactions (mpesa_transaction_id) WHERE mpesa_transaction_id IS NOT NULL;
CREATE UNIQUE INDEX wallet_transactions_payment_hash_unique_idx ON wallet_transactions (payment_hash) WHERE payment_hash IS NOT NULL;
CREATE INDEX wallet_transactions_user_created_at_idx ON wallet_transactions (user_id, created_at DESC);
CREATE INDEX wallet_transactions_status_idx ON wallet_transactions (status);

CREATE TABLE payment_audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES wallet_transactions(id) ON DELETE CASCADE,
    action VARCHAR(100) NOT NULL,
    actor VARCHAR(100) NOT NULL,
    provider_reference VARCHAR(255),
    details JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX payment_audit_events_transaction_id_idx ON payment_audit_events (transaction_id, created_at);

CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX password_reset_tokens_user_id_idx ON password_reset_tokens (user_id);
CREATE INDEX password_reset_tokens_expires_at_idx ON password_reset_tokens (expires_at);

CREATE TRIGGER counties_set_updated_at BEFORE UPDATE ON counties FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER listings_set_updated_at BEFORE UPDATE ON listings FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER orders_set_updated_at BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER invoices_set_updated_at BEFORE UPDATE ON invoices FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER wallet_transactions_set_updated_at BEFORE UPDATE ON wallet_transactions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
