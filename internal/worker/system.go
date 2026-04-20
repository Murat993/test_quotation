package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/murat/quotation-service/internal/usecase"
)

type Config struct {
	Interval           time.Duration
	Workers            int
	PollBatchSize      int
	BufferSize         int
	StaleAfter         time.Duration
	JobTimeout         time.Duration
	ResetStaleInterval time.Duration
}

type QuotationSystem struct {
	logger  *slog.Logger
	jobs    chan Job
	poller  *Poller
	pool    *Pool
	cleaner *StaleCleaner
}

func NewQuotationSystem(
	repo usecase.QuoteRepository,
	provider usecase.RateProvider,
	logger *slog.Logger,
	cfg Config,
) *QuotationSystem {
	jobs := make(chan Job, cfg.BufferSize)

	processor := NewQuotationProcessor(repo, provider, logger, cfg.JobTimeout)

	poller := NewPoller(
		repo,
		logger,
		cfg.Interval,
		cfg.PollBatchSize,
		jobs,
	)

	pool := NewPool(
		logger,
		cfg.Workers,
		jobs,
		processor.Handle,
	)

	cleaner := NewStaleCleaner(
		repo,
		logger,
		cfg.ResetStaleInterval,
		cfg.StaleAfter,
	)

	return &QuotationSystem{
		logger:  logger,
		jobs:    jobs,
		poller:  poller,
		pool:    pool,
		cleaner: cleaner,
	}
}

func (s *QuotationSystem) Start(ctx context.Context) error {
	s.logger.Info("Starting quotation system")

	s.pool.Start(ctx)

	go func() {
		if err := s.cleaner.Start(ctx); err != nil {
			s.logger.Error("Stale cleaner error", "error", err)
		}
	}()

	err := s.poller.Start(ctx)

	s.logger.Info("Quotation system stopping...")
	close(s.jobs)
	s.pool.Wait()
	s.logger.Info("Quotation system stopped")

	return err
}
