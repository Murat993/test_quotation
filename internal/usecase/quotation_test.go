package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/murat/quotation-service/internal/domain"
)

type mockRepo struct {
	saveRequestFunc                func(ctx context.Context, req *domain.QuoteRequest) error
	getRequestByIDFunc             func(ctx context.Context, id string) (*domain.QuoteRequest, error)
	getRequestByIdempotencyKeyFunc func(ctx context.Context, key string) (*domain.QuoteRequest, error)
	claimPendingRequestsFunc       func(ctx context.Context, limit int) ([]*domain.QuoteRequest, error)
	resetStaleRequestsFunc         func(ctx context.Context, olderThan time.Duration) (int64, error)
	updateStatusFunc               func(ctx context.Context, id string, oldStatus, newStatus domain.QuoteStatus, price float64, errMsg string) error
	saveLatestFunc                 func(ctx context.Context, quote *domain.LatestQuote) error
	getLatestFunc                  func(ctx context.Context, pair string) (*domain.LatestQuote, error)
}

func (m *mockRepo) SaveRequest(ctx context.Context, req *domain.QuoteRequest) error {
	return m.saveRequestFunc(ctx, req)
}

func (m *mockRepo) GetRequestByID(ctx context.Context, id string) (*domain.QuoteRequest, error) {
	return m.getRequestByIDFunc(ctx, id)
}

func (m *mockRepo) GetRequestByIdempotencyKey(ctx context.Context, key string) (*domain.QuoteRequest, error) {
	return m.getRequestByIdempotencyKeyFunc(ctx, key)
}

func (m *mockRepo) ClaimPendingRequests(ctx context.Context, limit int) ([]*domain.QuoteRequest, error) {
	return m.claimPendingRequestsFunc(ctx, limit)
}

func (m *mockRepo) ResetStaleRequests(ctx context.Context, olderThan time.Duration) (int64, error) {
	return m.resetStaleRequestsFunc(ctx, olderThan)
}

func (m *mockRepo) UpdateStatus(ctx context.Context, id string, oldStatus, newStatus domain.QuoteStatus, price float64, errMsg string) error {
	return m.updateStatusFunc(ctx, id, oldStatus, newStatus, price, errMsg)
}

func (m *mockRepo) SaveLatest(ctx context.Context, quote *domain.LatestQuote) error {
	return m.saveLatestFunc(ctx, quote)
}

func (m *mockRepo) GetLatest(ctx context.Context, pair string) (*domain.LatestQuote, error) {
	return m.getLatestFunc(ctx, pair)
}

func TestQuotation_RequestUpdate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			saveRequestFunc: func(ctx context.Context, req *domain.QuoteRequest) error {
				if req.Pair != "EUR/USD" {
					t.Errorf("expected pair EUR/USD, got %s", req.Pair)
				}
				return nil
			},
		}
		u := NewQuotation(repo)
		id, err := u.RequestUpdate(t.Context(), "EUR/USD", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == "" {
			t.Error("expected non-empty id")
		}
	})

	t.Run("invalid pair", func(t *testing.T) {
		u := NewQuotation(&mockRepo{})
		_, err := u.RequestUpdate(t.Context(), "INVALID", "")
		if !errors.Is(err, domain.ErrInvalidPair) {
			t.Errorf("expected error %v, got %v", domain.ErrInvalidPair, err)
		}
	})

	t.Run("idempotency success", func(t *testing.T) {
		key := "test-key"
		existingReq := &domain.QuoteRequest{ID: "existing-id", Pair: "EUR/USD", IdempotencyKey: key}
		repo := &mockRepo{
			getRequestByIdempotencyKeyFunc: func(ctx context.Context, k string) (*domain.QuoteRequest, error) {
				if k == key {
					return existingReq, nil
				}
				return nil, nil
			},
		}
		u := NewQuotation(repo)
		id, err := u.RequestUpdate(t.Context(), "EUR/USD", key)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != existingReq.ID {
			t.Errorf("expected id %s, got %s", existingReq.ID, id)
		}
	})
}

func TestQuotation_GetByRequestID(t *testing.T) {
	expectedReq := &domain.QuoteRequest{ID: "test-id", Pair: "EURUSD"}
	repo := &mockRepo{
		getRequestByIDFunc: func(ctx context.Context, id string) (*domain.QuoteRequest, error) {
			if id == "test-id" {
				return expectedReq, nil
			}
			return nil, errors.New("not found")
		},
	}
	u := NewQuotation(repo)

	t.Run("found", func(t *testing.T) {
		req, err := u.GetByRequestID(t.Context(), "test-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req != expectedReq {
			t.Errorf("expected %v, got %v", expectedReq, req)
		}
	})
}

func TestQuotation_GetLatest(t *testing.T) {
	expectedQuote := &domain.LatestQuote{Pair: "EUR/USD", Price: 1.1}
	repo := &mockRepo{
		getLatestFunc: func(ctx context.Context, pair string) (*domain.LatestQuote, error) {
			if pair == "EUR/USD" {
				return expectedQuote, nil
			}
			return nil, nil
		},
	}
	u := NewQuotation(repo)

	t.Run("success", func(t *testing.T) {
		quote, err := u.GetLatest(t.Context(), "EUR/USD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if quote != expectedQuote {
			t.Errorf("expected %v, got %v", expectedQuote, quote)
		}
	})

	t.Run("invalid pair", func(t *testing.T) {
		_, err := u.GetLatest(t.Context(), "INVALID")
		if !errors.Is(err, domain.ErrInvalidPair) {
			t.Errorf("expected error %v, got %v", domain.ErrInvalidPair, err)
		}
	})
}
