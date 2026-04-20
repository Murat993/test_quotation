-- +goose Up
ALTER TABLE quote_requests ADD COLUMN idempotency_key VARCHAR(255) NOT NULL DEFAULT '';
CREATE UNIQUE INDEX idx_quote_requests_idempotency_key ON quote_requests(idempotency_key) WHERE idempotency_key <> '';

-- +goose Down
DROP INDEX idx_quote_requests_idempotency_key;
ALTER TABLE quote_requests DROP COLUMN idempotency_key;
