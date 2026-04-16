package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/murat/quotation-service/config"
	"github.com/murat/quotation-service/internal/app"
)

// @title           Quotation Service API
// @version         1.0
// @description     API for fetching and storing exchange rate quotations.
// @host            localhost:8080
// @BasePath        /

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	application, err := app.New(&app.RunOpts{
		Config: cfg,
		Logger: logger,
	})
	if err != nil {
		logger.Error("failed to create application", "error", err)
		os.Exit(1)
	}

	if err := application.Run(); err != nil {
		logger.Error("application finished with error", "error", err)
		os.Exit(1)
	}
}
