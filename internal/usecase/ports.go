package usecase

import (
	"context"
	"time"

	"github.com/murat/quotation-service/internal/domain"
)

type QuoteRepository interface {
	SaveRequest(ctx context.Context, req *domain.QuoteRequest) error
	GetRequestByID(ctx context.Context, id string) (*domain.QuoteRequest, error)
	GetRequestByIdempotencyKey(ctx context.Context, key string) (*domain.QuoteRequest, error)
	ClaimPendingRequests(ctx context.Context, limit int) ([]*domain.QuoteRequest, error)
	ResetStaleRequests(ctx context.Context, olderThan time.Duration) (int64, error)
	UpdateStatus(ctx context.Context, id string, oldStatus, newStatus domain.QuoteStatus, price float64, errMsg string) error

	SaveLatest(ctx context.Context, quote *domain.LatestQuote) error
	GetLatest(ctx context.Context, pair string) (*domain.LatestQuote, error)
}

type RateProvider interface {
	GetRate(ctx context.Context, pair string) (float64, error)
}
