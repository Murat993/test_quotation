package worker

import (
	"log/slog"
	"sync"
)

type Handler func(workerID int, job Job)

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

func (p *Pool) Start() {
	for i := range p.workers {
		p.wg.Add(1)
		go p.run(i + 1)
	}
}

func (p *Pool) Wait() {
	p.wg.Wait()
}

func (p *Pool) run(id int) {
	defer p.wg.Done()

	p.logger.Info("Worker started", "worker_id", id)
	for job := range p.jobs {
		p.handler(id, job)
	}
	p.logger.Info("Worker stopping", "worker_id", id)
}
