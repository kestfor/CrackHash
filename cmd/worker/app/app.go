package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/kestfor/CrackHash/cmd/worker/handler"
	"github.com/kestfor/CrackHash/internal/services/worker"
	"github.com/kestfor/CrackHash/internal/services/worker/notifier"
	"github.com/kestfor/CrackHash/internal/services/worker/registerer"
	"github.com/kestfor/CrackHash/internal/services/worker/workerservice"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func New() *cobra.Command {
	var cfgPath string

	rootCmd := &cobra.Command{
		Use:           "worker service",
		Short:         "Worker service instance for cracking hashes",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cfgPath)
		},
	}

	rootCmd.Flags().
		StringVarP(&cfgPath, "config", "c", "configs/worker.yaml", "path to configuration file")

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

	slog.Info("config parsed")

	if err := cfg.Validate(); err != nil {
		slog.Error("validate config failed", slog.Any("error", err))
		return err
	}

	slog.Info("config validated")

	slog.Info("registering worker in manager...")
	rgr := registerer.NewHTTPRegisterer(cfg.Registerer)
	id, err := rgr.Register()
	if err != nil {
		slog.Error("registering failed", slog.Any("error", err))
		return err
	}

	slog.SetDefault(slog.With(
		slog.String("service", "crackhash-worker"),
		slog.Any("worker_id", id),
	))

	slog.Info("registered in manager")
	slog.Info("initializing dependencies...")

	// Set notifier's self port from HTTP config
	cfg.Notifier.SelfPort = cfg.HTTP.Port
	httpNotifier := notifier.NewHTTPNotifier(cfg.Notifier)
	workerService := workerservice.NewService(cfg.Worker, httpNotifier)
	slog.Info("dependencies initialized")

	initServer(cfg.HTTP, workerService)

	return nil
}

func initServer(httpServerConfig *HTTPServerConfig, workerService worker.Service) {
	slog.Info("initializing http server...")

	workerHandler := handler.NewHandler(workerService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/tasks/", workerHandler.HandleCreateTask)
	mux.HandleFunc("GET /api/v1/tasks/{task_id}/progress", workerHandler.HandleGetProgress)
	mux.HandleFunc("PUT /api/v1/tasks/{task_id}/do", workerHandler.HandleDoTask)
	mux.HandleFunc("DELETE /api/v1/tasks/{task_id}", workerHandler.HandleDeleteTask)
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
