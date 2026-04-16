package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/murat/quotation-service/internal/domain"
	"github.com/murat/quotation-service/internal/usecase"
)

type QuotationProcessor struct {
	repo     usecase.QuoteRepository
	provider usecase.RateProvider
	logger   *slog.Logger
	timeout  time.Duration
}

func NewQuotationProcessor(
	repo usecase.QuoteRepository,
	provider usecase.RateProvider,
	logger *slog.Logger,
	timeout time.Duration,
) *QuotationProcessor {
	return &QuotationProcessor{
		repo:     repo,
		provider: provider,
		logger:   logger,
		timeout:  timeout,
	}
}

func (p *QuotationProcessor) Handle(workerID int, job Job) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	p.logger.Info("Processing job", "worker_id", workerID, "id", job.ID, "pair", job.Pair)

	rate, status, errMsg := p.fetchRate(ctx, workerID, job)

	if err := p.repo.UpdateStatus(ctx, job.ID, domain.StatusProcessing, status, rate, errMsg); err != nil {
		p.logger.Error("Failed to update request status", "worker_id", workerID, "id", job.ID, "error", err)
		return
	}

	if status == domain.StatusDone {
		latest := &domain.LatestQuote{
			Pair:      job.Pair,
			Price:     rate,
			UpdatedAt: time.Now(),
		}
		if err := p.repo.SaveLatest(ctx, latest); err != nil {
			p.logger.Error("Failed to save latest quote", "worker_id", workerID, "pair", job.Pair, "error", err)
		}
	}
}

func (p *QuotationProcessor) fetchRate(ctx context.Context, workerID int, job Job) (float64, domain.QuoteStatus, string) {
	rate, err := p.provider.GetRate(ctx, job.Pair)
	if err != nil {
		p.logger.Error("Failed to fetch rate", "worker_id", workerID, "id", job.ID, "pair", job.Pair, "error", err)
		return 0, domain.StatusFailed, err.Error()
	}

	p.logger.Info("Fetched rate", "worker_id", workerID, "id", job.ID, "pair", job.Pair, "rate", rate)
	return rate, domain.StatusDone, ""
}
