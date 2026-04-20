package usecase

import (
	"context"
	"errors"

	"github.com/murat/quotation-service/internal/domain"
)

var (
	ErrInvalidPair  = domain.ErrInvalidPair
	ErrDuplicateKey = errors.New("duplicate idempotency key")
)

type QuotationUseCase interface {
	RequestUpdate(ctx context.Context, pair string, idempotencyKey string) (string, error)
	GetByRequestID(ctx context.Context, id string) (*domain.QuoteRequest, error)
	GetLatest(ctx context.Context, pair string) (*domain.LatestQuote, error)
}

type Quotation struct {
	repo QuoteRepository
}

func NewQuotation(repo QuoteRepository) *Quotation {
	return &Quotation{repo: repo}
}

func (u *Quotation) RequestUpdate(ctx context.Context, pair string, idempotencyKey string) (string, error) {
	if idempotencyKey != "" {
		existing, err := u.repo.GetRequestByIdempotencyKey(ctx, idempotencyKey)
		if err != nil {
			return "", err
		}
		if existing != nil {
			return existing.ID, nil
		}
	}

	req, err := domain.NewQuoteRequest(pair, idempotencyKey)
	if err != nil {
		return "", err
	}

	if err := u.repo.SaveRequest(ctx, req); err != nil {
		return "", err
	}

	return req.ID, nil
}

func (u *Quotation) GetByRequestID(ctx context.Context, id string) (*domain.QuoteRequest, error) {
	return u.repo.GetRequestByID(ctx, id)
}

func (u *Quotation) GetLatest(ctx context.Context, pair string) (*domain.LatestQuote, error) {
	normalized, err := domain.NormalizePair(pair)
	if err != nil {
		return nil, err
	}
	return u.repo.GetLatest(ctx, normalized)
}
