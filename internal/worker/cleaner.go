package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/murat/quotation-service/internal/usecase"
)

type StaleCleaner struct {
	repo       usecase.QuoteRepository
	logger     *slog.Logger
	interval   time.Duration
	staleAfter time.Duration
}

func NewStaleCleaner(
	repo usecase.QuoteRepository,
	logger *slog.Logger,
	interval time.Duration,
	staleAfter time.Duration,
) *StaleCleaner {
	return &StaleCleaner{
		repo:       repo,
		logger:     logger,
		interval:   interval,
		staleAfter: staleAfter,
	}
}

func (c *StaleCleaner) Start(ctx context.Context) error {
	c.logger.Info("Starting stale cleaner", "interval", c.interval, "stale_after", c.staleAfter)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stale cleaner stopping...")
			return nil
		case <-ticker.C:
			c.reset(ctx)
		}
	}
}

func (c *StaleCleaner) reset(ctx context.Context) {
	affected, err := c.repo.ResetStaleRequests(ctx, c.staleAfter)
	if err != nil {
		c.logger.Error("Failed to reset stale requests", "error", err)
		return
	}
	if affected > 0 {
		c.logger.Info("Reset stale requests", "count", affected)
	}
}
