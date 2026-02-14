package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/kestfor/CrackHash/cmd/manager/handler"
	"github.com/kestfor/CrackHash/internal/services/manager"
	"github.com/kestfor/CrackHash/internal/services/manager/healthchecker"
	"github.com/kestfor/CrackHash/internal/services/manager/service"
	"github.com/kestfor/CrackHash/pkg/logging"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func New() *cobra.Command {
	var cfgPath string

	rootCmd := &cobra.Command{
		Use:           "manager service",
		Short:         "Manager service for cracking hashes",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cfgPath)
		},
	}

	rootCmd.Flags().
		StringVarP(&cfgPath, "config", "c", "configs/manager.yaml", "path to configuration file")

	return rootCmd
}

func run(cfgPath string) error {
	slog.Info("parsing config...")

	bytes, err := os.ReadFile(cfgPath)
	if err != nil {
		slog.Error("read file failed", slog.Any("error", err))
		return err
	}

	cfg := &Config{}

	if err := yaml.Unmarshal(bytes, cfg); err != nil {
		slog.Error("parse config failed", slog.Any("error", err))
		return err
	}

	logging.InitLogger(cfg.Logger, slog.String("service", "manager"))

	slog.Info("config parsed")

	if err := cfg.Validate(); err != nil {
		slog.Error("validate config failed", slog.Any("error", err))
		return err
	}

	slog.Info("config validated")

	slog.Info("registering worker in manager...")

	slog.Info("initializing dependencies...")
	slog.Info("dependencies initialized")

	var healthCheckProvider = func(workerAddr string) healthchecker.HealthChecker {
		return healthCheck(workerAddr, cfg)
	}

	managerService := service.NewService(cfg.HashCracker.Alphabet, healthCheckProvider)
	initServer(cfg.HTTP, managerService)

	return nil
}

func healthCheck(workerAddr string, config *Config) healthchecker.HealthChecker {
	return healthchecker.NewHTTPHealthChecker(&healthchecker.HTTPHealthCheckerConfig{
		URL:      fmt.Sprintf("http://%s/health", workerAddr),
		MaxTries: config.Healthcheck.MaxTries,
		Period:   config.Healthcheck.Period,
	})
}

func initServer(httpServerConfig *HTTPConfig, managerService manager.Service) {
	slog.Info("initializing http server...")

	workerHandler := handler.NewHandler(managerService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/hash/crack", workerHandler.HandleCreateTask)
	mux.HandleFunc("GET /api/hash/status", workerHandler.HandleGetTaskProgress)
	mux.HandleFunc("POST /api/tasks/progress", workerHandler.HandleUpdateProgress)
	mux.HandleFunc("GET /api/hash/register-worker", workerHandler.HandleRegisterWorker)
	mux.HandleFunc("GET /health", healthHandler)

	wrappedMux := recoverMiddleware(mux)

	slog.Info("http server initialized, serving...")
	err := http.ListenAndServe(fmt.Sprintf(":%d", httpServerConfig.Port), wrappedMux)
	if err != nil {
		slog.Error("failed to start server", "error", err)
	}

}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				slog.Error("Unexpected panic", slog.Any("error", err), slog.String("stacktrace", string(debug.Stack())))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
