package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/murat/quotation-service/internal/domain"
)

type quoteRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewQuoteRepository(db *sql.DB, logger *slog.Logger) *quoteRepository {
	return &quoteRepository{db: db, logger: logger}
}

func (r *quoteRepository) SaveRequest(ctx context.Context, req *domain.QuoteRequest) error {
	r.logger.Info("Saving quote request", "id", req.ID, "pair", req.Pair, "idempotency_key", req.IdempotencyKey)

	query := `INSERT INTO quote_requests (id, pair, status, price, error, idempotency_key, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query, req.ID, req.Pair, req.Status, req.Price, req.Error, req.IdempotencyKey, req.CreatedAt, req.UpdatedAt)
	return err
}

func (r *quoteRepository) GetRequestByID(ctx context.Context, id string) (*domain.QuoteRequest, error) {
	r.logger.Info("Getting quote request by ID", "id", id)
	query := `SELECT id, pair, status, price, error, idempotency_key, created_at, updated_at 
			  FROM quote_requests WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	return scanQuoteRequest(row)
}

func (r *quoteRepository) GetRequestByIdempotencyKey(ctx context.Context, key string) (*domain.QuoteRequest, error) {
	r.logger.Info("Getting quote request by idempotency key", "key", key)
	query := `SELECT id, pair, status, price, error, idempotency_key, created_at, updated_at 
			  FROM quote_requests WHERE idempotency_key = $1`
	row := r.db.QueryRowContext(ctx, query, key)

	return scanQuoteRequest(row)
}

func (r *quoteRepository) ClaimPendingRequests(ctx context.Context, limit int) ([]*domain.QuoteRequest, error) {
	const query = `
UPDATE quote_requests
SET status = $3, 
    updated_at = NOW()
WHERE id IN (
	SELECT id 
	FROM quote_requests 
	WHERE status = $1 
	ORDER BY created_at ASC 
	LIMIT $2 
	FOR UPDATE SKIP LOCKED
)
RETURNING id, pair, status, price, error, idempotency_key, created_at, updated_at;
`
	rows, err := r.db.QueryContext(ctx, query, domain.StatusPending, limit, domain.StatusProcessing)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanQuoteRequests(rows)
}

func scanQuoteRequests(rows *sql.Rows) ([]*domain.QuoteRequest, error) {
	var reqs []*domain.QuoteRequest
	for rows.Next() {
		req, err := scanQuoteRequest(rows)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reqs, nil
}

func (r *quoteRepository) ResetStaleRequests(ctx context.Context, olderThan time.Duration) (int64, error) {
	r.logger.Info("Resetting stale requests", "olderThan", olderThan)
	query := `UPDATE quote_requests SET status = $1, updated_at = NOW() 
			  WHERE status = $2 AND updated_at < NOW() - $3::interval`
	res, err := r.db.ExecContext(ctx, query, domain.StatusPending, domain.StatusProcessing, olderThan.String())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *quoteRepository) UpdateStatus(ctx context.Context, id string, oldStatus, newStatus domain.QuoteStatus, price float64, errMsg string) error {
	r.logger.Info("Updating quote request status", "id", id, "old", oldStatus, "new", newStatus)
	query := `UPDATE quote_requests SET status = $1, price = $2, error = $3, updated_at = NOW() 
			  WHERE id = $4 AND status = $5`
	res, err := r.db.ExecContext(ctx, query, newStatus, price, errMsg, id, oldStatus)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("conflict: quote request not in expected state")
	}

	return nil
}

func (r *quoteRepository) SaveLatest(ctx context.Context, quote *domain.LatestQuote) error {
	r.logger.Info("Saving latest quote", "pair", quote.Pair, "price", quote.Price)
	query := `INSERT INTO latest_quotes (pair, price, updated_at) VALUES ($1, $2, $3)
			  ON CONFLICT (pair) DO UPDATE SET price = $2, updated_at = $3`
	_, err := r.db.ExecContext(ctx, query, quote.Pair, quote.Price, quote.UpdatedAt)
	return err
}

func (r *quoteRepository) GetLatest(ctx context.Context, pair string) (*domain.LatestQuote, error) {
	r.logger.Info("Getting latest quote", "pair", pair)
	query := `SELECT pair, price, updated_at FROM latest_quotes WHERE pair = $1`
	row := r.db.QueryRowContext(ctx, query, pair)

	var q domain.LatestQuote
	err := row.Scan(&q.Pair, &q.Price, &q.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &q, nil
}

func scanQuoteRequest(scanner interface {
	Scan(dest ...any) error
}) (*domain.QuoteRequest, error) {
	var req domain.QuoteRequest
	if err := scanner.Scan(
		&req.ID,
		&req.Pair,
		&req.Status,
		&req.Price,
		&req.Error,
		&req.IdempotencyKey,
		&req.CreatedAt,
		&req.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &req, nil
}
