package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/murat/quotation-service/config"
	delivery "github.com/murat/quotation-service/internal/delivery/http"
	"github.com/murat/quotation-service/internal/provider/exchangerate"
	"github.com/murat/quotation-service/internal/repository/postgres"
	"github.com/murat/quotation-service/internal/usecase"
	"github.com/murat/quotation-service/internal/worker"
	pkgpg "github.com/murat/quotation-service/pkg/postgres"
)

type RunOpts struct {
	Config *config.Config
	Logger *slog.Logger
}

type App struct {
	cfg *config.Config
	l   *slog.Logger

	db       *sql.DB
	server   *http.Server
	quotWork *worker.QuotationSystem
}

func New(opts *RunOpts) (*App, error) {
	if opts == nil {
		return nil, errors.New("opts is nil")
	}
	if opts.Config == nil {
		return nil, errors.New("config is nil")
	}
	if opts.Logger == nil {
		return nil, errors.New("logger is nil")
	}

	return &App{
		cfg: opts.Config,
		l:   opts.Logger,
	}, nil
}

func (app *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.initServices(); err != nil {
		return err
	}
	defer app.db.Close()

	return app.waitShutdown(ctx, stop)
}

func (app *App) initServices() error {
	db, err := pkgpg.New(pkgpg.Config{
		URL:             app.cfg.DBURL,
		MaxOpenConns:    app.cfg.DBMaxOpenConns,
		MaxIdleConns:    app.cfg.DBMaxIdleConns,
		ConnMaxLifetime: app.cfg.DBConnMaxLifetime,
		ConnMaxIdleTime: app.cfg.DBConnMaxIdleTime,
		PingTimeout:     app.cfg.DBPingTimeout,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	app.db = db

	repo := postgres.NewQuoteRepository(db, app.l)
	provider := exchangerate.NewProvider(app.cfg.ExchangeRateAPIKey)
	uc := usecase.NewQuotation(repo)

	app.quotWork = worker.NewQuotationSystem(
		repo,
		provider,
		app.l,
		worker.Config{
			Interval:           app.cfg.WorkerInterval,
			Workers:            app.cfg.WorkerPoolSize,
			PollBatchSize:      app.cfg.PollBatchSize,
			BufferSize:         app.cfg.JobsChannelSize,
			StaleAfter:         app.cfg.WorkerStaleAfter,
			JobTimeout:         app.cfg.WorkerJobTimeout,
			ResetStaleInterval: app.cfg.ResetStaleInterval,
		},
	)

	handler := delivery.NewQuoteHandler(uc, app.l)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	app.server = &http.Server{
		Addr:         ":" + app.cfg.Port,
		Handler:      mux,
		ReadTimeout:  app.cfg.ReadTimeout,
		WriteTimeout: app.cfg.WriteTimeout,
	}

	return nil
}

func (app *App) waitShutdown(ctx context.Context, stop context.CancelFunc) error {
	errCh := make(chan error, 2)

	go func() {
		if err := app.quotWork.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			app.l.Error("worker error", "error", err)
			errCh <- err
		}
	}()

	go func() {
		app.l.Info("Starting server", "port", app.cfg.Port)
		if err := app.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.l.Error("server error", "error", err)
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		app.l.Info("Shutdown signal received, shutting down...")
	case err := <-errCh:
		app.l.Error("Critical error, shutting down", "error", err)
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), app.cfg.ShutdownTimeout)
	defer cancel()

	if err := app.server.Shutdown(shutdownCtx); err != nil {
		app.l.Error("server shutdown error", "error", err)
		return err
	}

	app.l.Info("Shutdown completed")
	return nil
}
