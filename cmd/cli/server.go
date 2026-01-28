package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"kii.com/internal/application/usecase"
	"kii.com/internal/infrastructure/config"
	httphandler "kii.com/internal/infrastructure/http"
	"kii.com/internal/infrastructure/logger"
	"kii.com/internal/infrastructure/repository"
	"kii.com/internal/infrastructure/validator"

	"github.com/spf13/cobra"
)

const serverDir = "server"

var apiServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Run API Server.",
	RunE: func(_ *cobra.Command, _ []string) error {
		// Initialize logger
		appLogger := logger.NewLogger()

		// Get config directory (relative to where the binary is run from)
		configDir := filepath.Join("cmd", "config", serverDir)
		if _, err := os.Stat(configDir); os.IsNotExist(err) {
			// Try absolute path from project root
			configDir = filepath.Join(".", "cmd", "config", serverDir)
		}

		// Load configuration
		cfg, err := config.LoadConfig(configDir)
		if err != nil {
			appLogger.LogError(context.TODO(), "Failed to load config", err)
			return fmt.Errorf("failed to load config: %w", err)
		}

		appLogger.LogInfo(context.TODO(), "Configuration loaded",
			"port", cfg.Server.Port,
			"timestamp_tolerance", cfg.Webhook.TimestampTolerance.String())

		// Initialize infrastructure adapters
		ledgerRepo := repository.NewInMemoryLedger(appLogger)
		webhookValidator := validator.NewHMACValidator(
			cfg.Webhook.HMACSecret,
			cfg.Webhook.TimestampTolerance,
			appLogger,
		)

		// Initialize use cases
		processWebhookUseCase := usecase.NewProcessWebhookUseCase(
			webhookValidator,
			ledgerRepo,
		)
		getBalanceUseCase := usecase.NewGetBalanceUseCase(ledgerRepo)

		// Initialize HTTP handler
		handler := httphandler.NewHandler(
			processWebhookUseCase,
			getBalanceUseCase,
			webhookValidator,
			appLogger,
		)

		// Setup routes
		mux := handler.SetupRoutes()

		// Create HTTP server
		addr := ":" + cfg.Server.Port
		server := &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		// Channel to capture termination signals
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

		// Error channel to capture errors from server
		errChan := make(chan error, 1)

		// Start server in a goroutine
		go func() {
			appLogger.LogInfo(context.TODO(), "Starting server",
				"address", addr,
				"timestamp_tolerance", cfg.Webhook.TimestampTolerance.String())
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errChan <- err
			}
		}()

		// Graceful shutdown
		select {
		case <-signalChan:
			appLogger.LogInfo(context.TODO(), "Received termination signal. Initiating graceful shutdown...")

			// Create shutdown context with timeout
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := server.Shutdown(shutdownCtx); err != nil {
				appLogger.LogError(context.TODO(), "Server forced to shutdown", err)
				return err
			}

			appLogger.LogInfo(context.TODO(), "Server stopped gracefully")
		case err := <-errChan:
			appLogger.LogError(context.TODO(), "Server error", err)
			return err
		}

		return nil
	},
}

func init() { //nolint:gochecknoinits
	rootCmd.AddCommand(apiServerCmd)
}
