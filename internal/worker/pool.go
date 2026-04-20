package worker

import (
	"context"
	"log/slog"
	"sync"
)

type Handler func(ctx context.Context, workerID int, job Job)

type Pool struct {
	logger  *slog.Logger
	workers int
	jobs    <-chan Job
	handler Handler
	wg      sync.WaitGroup
}

func NewPool(logger *slog.Logger, workers int, jobs <-chan Job, handler Handler) *Pool {
	return &Pool{
		logger:  logger,
		workers: workers,
		jobs:    jobs,
		handler: handler,
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := range p.workers {
		p.wg.Add(1)
		go p.run(ctx, i+1)
	}
}

func (p *Pool) Wait() {
	p.wg.Wait()
}

func (p *Pool) run(ctx context.Context, id int) {
	defer p.wg.Done()

	p.logger.Info("Worker started", "worker_id", id)
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Worker stopping (context canceled)", "worker_id", id)
			return
		case job, ok := <-p.jobs:
			if !ok {
				p.logger.Info("Worker stopping (jobs channel closed)", "worker_id", id)
				return
			}
			p.handler(ctx, id, job)
		}
	}
}
