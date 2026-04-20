package worker

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/murat/quotation-service/internal/domain"
)

type mockRepo struct {
	claimFunc        func(ctx context.Context, limit int) ([]*domain.QuoteRequest, error)
	updateStatusFunc func(ctx context.Context, id string, oldStatus, newStatus domain.QuoteStatus, price float64, errMsg string) error
	saveLatestFunc   func(ctx context.Context, quote *domain.LatestQuote) error
	resetStaleFunc   func(ctx context.Context, olderThan time.Duration) (int64, error)
}

func (m *mockRepo) ClaimPendingRequests(ctx context.Context, limit int) ([]*domain.QuoteRequest, error) {
	return m.claimFunc(ctx, limit)
}
func (m *mockRepo) ResetStaleRequests(ctx context.Context, olderThan time.Duration) (int64, error) {
	if m.resetStaleFunc != nil {
		return m.resetStaleFunc(ctx, olderThan)
	}
	return 0, nil
}
func (m *mockRepo) UpdateStatus(ctx context.Context, id string, oldStatus, newStatus domain.QuoteStatus, price float64, errMsg string) error {
	return m.updateStatusFunc(ctx, id, oldStatus, newStatus, price, errMsg)
}
func (m *mockRepo) SaveLatest(ctx context.Context, quote *domain.LatestQuote) error {
	return m.saveLatestFunc(ctx, quote)
}
func (m *mockRepo) SaveRequest(ctx context.Context, req *domain.QuoteRequest) error { return nil }
func (m *mockRepo) GetRequestByID(ctx context.Context, id string) (*domain.QuoteRequest, error) {
	return nil, nil
}
func (m *mockRepo) GetRequestByIdempotencyKey(ctx context.Context, key string) (*domain.QuoteRequest, error) {
	return nil, nil
}
func (m *mockRepo) GetLatest(ctx context.Context, pair string) (*domain.LatestQuote, error) {
	return nil, nil
}

type mockProvider struct {
	getRateFunc func(ctx context.Context, pair string) (float64, error)
}

func (m *mockProvider) GetRate(ctx context.Context, pair string) (float64, error) {
	return m.getRateFunc(ctx, pair)
}

func TestQuotationSystem(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("Full success flow", func(t *testing.T) {
		req := &domain.QuoteRequest{
			ID:     "1",
			Pair:   "EUR/USD",
			Status: domain.StatusPending,
		}

		var mu sync.Mutex
		updatedStatuses := make([]domain.QuoteStatus, 0)
		repo := &mockRepo{
			claimFunc: func(ctx context.Context, limit int) ([]*domain.QuoteRequest, error) {
				return []*domain.QuoteRequest{req}, nil
			},
			updateStatusFunc: func(ctx context.Context, id string, old, new domain.QuoteStatus, price float64, errMsg string) error {
				mu.Lock()
				defer mu.Unlock()
				updatedStatuses = append(updatedStatuses, new)
				req.Status = new
				req.Price = price
				return nil
			},
			saveLatestFunc: func(ctx context.Context, q *domain.LatestQuote) error {
				return nil
			},
		}

		provider := &mockProvider{
			getRateFunc: func(ctx context.Context, pair string) (float64, error) {
				return 1.1, nil
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		s := NewQuotationSystem(repo, provider, logger, Config{
			Interval:           10 * time.Millisecond,
			Workers:            1,
			PollBatchSize:      1,
			BufferSize:         1,
			StaleAfter:         time.Minute,
			JobTimeout:         time.Second,
			ResetStaleInterval: time.Minute,
		})

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		err := s.Start(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("expected nil error or canceled, got %v", err)
		}

		mu.Lock()
		defer mu.Unlock()
		if len(updatedStatuses) < 1 {
			t.Errorf("expected at least 1 status update call, got %d", len(updatedStatuses))
		}
		if req.Price != 1.1 {
			t.Errorf("expected price 1.1, got %f", req.Price)
		}
	})
}

func TestPool(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("Pool processes all jobs", func(t *testing.T) {
		jobs := make(chan Job, 10)
		processed := make(map[string]bool)
		var mu sync.Mutex

		handler := func(ctx context.Context, workerID int, job Job) {
			mu.Lock()
			processed[job.ID] = true
			mu.Unlock()
		}

		pool := NewPool(logger, 3, jobs, handler)
		pool.Start(context.Background())

		for i := 1; i <= 5; i++ {
			jobs <- Job{ID: string(rune('0' + i)), Pair: "EUR/USD"}
		}
		close(jobs)
		pool.Wait()

		if len(processed) != 5 {
			t.Errorf("expected 5 processed jobs, got %d", len(processed))
		}
	})
}

func TestProcessor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("Processor success", func(t *testing.T) {
		job := Job{ID: "1", Pair: "EUR/USD"}
		repo := &mockRepo{
			updateStatusFunc: func(ctx context.Context, id string, old, new domain.QuoteStatus, price float64, errMsg string) error {
				if id != "1" || old != domain.StatusProcessing || new != domain.StatusDone || price != 1.1 {
					t.Errorf("unexpected status update: id=%s, old=%s, new=%s, price=%f", id, old, new, price)
				}
				return nil
			},
			saveLatestFunc: func(ctx context.Context, quote *domain.LatestQuote) error {
				if quote.Pair != "EUR/USD" || quote.Price != 1.1 {
					t.Errorf("unexpected latest quote save: pair=%s, price=%f", quote.Pair, quote.Price)
				}
				return nil
			},
		}
		provider := &mockProvider{
			getRateFunc: func(ctx context.Context, pair string) (float64, error) {
				return 1.1, nil
			},
		}

		p := NewQuotationProcessor(repo, provider, logger, time.Second)
		p.Handle(context.Background(), 1, job)
	})

	t.Run("Processor failure", func(t *testing.T) {
		job := Job{ID: "2", Pair: "INVALID"}
		repo := &mockRepo{
			updateStatusFunc: func(ctx context.Context, id string, old, new domain.QuoteStatus, price float64, errMsg string) error {
				if id != "2" || new != domain.StatusFailed || errMsg != "provider error" {
					t.Errorf("unexpected status update for failure: id=%s, new=%s, err=%s", id, new, errMsg)
				}
				return nil
			},
		}
		provider := &mockProvider{
			getRateFunc: func(ctx context.Context, pair string) (float64, error) {
				return 0, errors.New("provider error")
			},
		}

		p := NewQuotationProcessor(repo, provider, logger, time.Second)
		p.Handle(context.Background(), 1, job)
	})
}

func TestPoller(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("Poller dispatches jobs", func(t *testing.T) {
		repo := &mockRepo{
			claimFunc: func(ctx context.Context, limit int) ([]*domain.QuoteRequest, error) {
				return []*domain.QuoteRequest{
					{ID: "1", Pair: "EUR/USD"},
				}, nil
			},
		}

		jobs := make(chan Job, 1)
		poller := NewPoller(repo, logger, 10*time.Millisecond, 1, jobs)

		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
		defer cancel()

		_ = poller.Start(ctx)

		select {
		case job := <-jobs:
			if job.ID != "1" {
				t.Errorf("expected job ID 1, got %s", job.ID)
			}
		default:
			t.Error("expected a job in channel")
		}
	})
}
