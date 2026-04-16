package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/murat/quotation-service/internal/usecase"
)

type Poller struct {
	repo       usecase.QuoteRepository
	logger     *slog.Logger
	interval   time.Duration
	batchSize  int
	staleAfter time.Duration
	out        chan<- Job
}

func NewPoller(
	repo usecase.QuoteRepository,
	logger *slog.Logger,
	interval time.Duration,
	batchSize int,
	staleAfter time.Duration,
	out chan<- Job,
) *Poller {
	return &Poller{
		repo:       repo,
		logger:     logger,
		interval:   interval,
		batchSize:  batchSize,
		staleAfter: staleAfter,
		out:        out,
	}
}

func (p *Poller) Start(ctx context.Context) error {
	p.logger.Info("Starting poller",
		"interval", p.interval,
		"batch_size", p.batchSize,
		"stale_after", p.staleAfter,
	)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			p.resetStale(ctx)
			p.poll(ctx)
		}
	}
}

func (p *Poller) resetStale(ctx context.Context) {
	if affected, err := p.repo.ResetStaleRequests(ctx, p.staleAfter); err != nil {
		p.logger.Error("Failed to reset stale requests", "error", err)
	} else if affected > 0 {
		p.logger.Info("Reset stale requests", "count", affected)
	}
}

func (p *Poller) poll(ctx context.Context) {
	reqs, err := p.repo.ClaimPendingRequests(ctx, p.batchSize)
	if err != nil {
		p.logger.Error("Failed to fetch and lock pending requests", "error", err)
		return
	}

	for _, req := range reqs {
		select {
		case <-ctx.Done():
			return
		case p.out <- Job{
			ID:   req.ID,
			Pair: req.Pair,
		}:
			p.logger.Debug("Job dispatched", "id", req.ID)
		}
	}
}
