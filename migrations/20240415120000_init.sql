-- +goose Up
CREATE TABLE IF NOT EXISTS quote_requests (
    id UUID PRIMARY KEY,
    pair VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    price DOUBLE PRECISION,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS latest_quotes (
    pair VARCHAR(20) PRIMARY KEY,
    price DOUBLE PRECISION NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS latest_quotes;
DROP TABLE IF EXISTS quote_requests;
