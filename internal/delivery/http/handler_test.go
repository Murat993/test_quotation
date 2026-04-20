package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/murat/quotation-service/internal/domain"
)

type mockUseCase struct {
	requestUpdateFunc  func(ctx context.Context, pair string, idempotencyKey string) (string, error)
	getByRequestIDFunc func(ctx context.Context, id string) (*domain.QuoteRequest, error)
	getLatestFunc      func(ctx context.Context, pair string) (*domain.LatestQuote, error)
}

func (m *mockUseCase) RequestUpdate(ctx context.Context, pair string, idempotencyKey string) (string, error) {
	return m.requestUpdateFunc(ctx, pair, idempotencyKey)
}

func (m *mockUseCase) GetByRequestID(ctx context.Context, id string) (*domain.QuoteRequest, error) {
	return m.getByRequestIDFunc(ctx, id)
}

func (m *mockUseCase) GetLatest(ctx context.Context, pair string) (*domain.LatestQuote, error) {
	if m.getLatestFunc != nil {
		return m.getLatestFunc(ctx, pair)
	}
	return nil, nil
}

func TestQuoteHandler_RequestUpdate(t *testing.T) {
	t.Run("Success without idempotency key", func(t *testing.T) {
		uc := &mockUseCase{
			requestUpdateFunc: func(ctx context.Context, pair string, key string) (string, error) {
				if key != "" {
					t.Errorf("expected empty idempotency key, got %v", key)
				}
				return "test-id", nil
			},
		}
		h := NewQuoteHandler(uc, nil)

		reqBody := `{"pair": "EUR/USD"}`
		req := httptest.NewRequest("POST", "/quotes/update", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()

		h.RequestUpdate(rr, req)

		if rr.Code != http.StatusAccepted {
			t.Errorf("expected status 202, got %d", rr.Code)
		}

		var resp requestUpdateResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		if resp.RequestID != "test-id" {
			t.Errorf("expected id test-id, got %s", resp.RequestID)
		}
	})

	t.Run("Success with idempotency key", func(t *testing.T) {
		expectedKey := "test-key"
		uc := &mockUseCase{
			requestUpdateFunc: func(ctx context.Context, pair string, key string) (string, error) {
				if key != expectedKey {
					t.Errorf("expected idempotency key %s, got %v", expectedKey, key)
				}
				return "test-id", nil
			},
		}
		h := NewQuoteHandler(uc, nil)

		reqBody := `{"pair": "EUR/USD"}`
		req := httptest.NewRequest("POST", "/quotes/update", strings.NewReader(reqBody))
		req.Header.Set("X-Idempotency-Key", expectedKey)
		rr := httptest.NewRecorder()

		h.RequestUpdate(rr, req)

		if rr.Code != http.StatusAccepted {
			t.Errorf("expected status 202, got %d", rr.Code)
		}
	})
}

func TestQuoteHandler_GetStatus(t *testing.T) {
	now := time.Now()
	uc := &mockUseCase{
		getByRequestIDFunc: func(ctx context.Context, id string) (*domain.QuoteRequest, error) {
			if id == "done-id" {
				return &domain.QuoteRequest{
					ID:        "done-id",
					Pair:      "EUR/USD",
					Status:    domain.StatusDone,
					Price:     1.1,
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			}
			return &domain.QuoteRequest{
				ID:        "pending-id",
				Pair:      "EUR/USD",
				Status:    domain.StatusPending,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}
	h := NewQuoteHandler(uc, nil)

	t.Run("Pending status (no price/error in JSON)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/quotes/requests/pending-id", nil)
		req.SetPathValue("id", "pending-id")
		rr := httptest.NewRecorder()

		h.GetStatus(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		body := rr.Body.String()
		if strings.Contains(body, `"price"`) {
			t.Error("expected price to be omitted for pending status")
		}
		if strings.Contains(body, `"error"`) {
			t.Error("expected error to be omitted for pending status")
		}
	})

	t.Run("Done status (price in JSON)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/quotes/requests/done-id", nil)
		req.SetPathValue("id", "done-id")
		rr := httptest.NewRecorder()

		h.GetStatus(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp quoteRequestResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		if resp.Price != 1.1 {
			t.Errorf("expected price 1.1, got %f", resp.Price)
		}
	})
}

func TestQuoteHandler_GetLatest(t *testing.T) {
	now := time.Now()
	uc := &mockUseCase{
		getLatestFunc: func(ctx context.Context, pair string) (*domain.LatestQuote, error) {
			if pair == "EUR/USD" {
				return &domain.LatestQuote{
					Pair:      "EUR/USD",
					Price:     1.1,
					UpdatedAt: now,
				}, nil
			}
			if pair == "INVALID" {
				return nil, domain.ErrInvalidPair
			}
			return nil, nil
		},
	}
	h := NewQuoteHandler(uc, nil)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/quotes/latest?pair=EUR/USD", nil)
		rr := httptest.NewRecorder()

		h.GetLatest(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp latestQuoteResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		if resp.Price != 1.1 {
			t.Errorf("expected price 1.1, got %f", resp.Price)
		}
	})

	t.Run("Not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/quotes/latest?pair=GBP/USD", nil)
		rr := httptest.NewRecorder()

		h.GetLatest(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rr.Code)
		}
	})

	t.Run("Invalid pair", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/quotes/latest?pair=INVALID", nil)
		rr := httptest.NewRecorder()

		h.GetLatest(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}
