package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/cmd/worker/handler"
	workersrv "github.com/kestfor/CrackHash/internal/services/worker"
	"github.com/kestfor/CrackHash/internal/services/worker/notifier"
	"github.com/kestfor/CrackHash/internal/services/worker/registerer"
	worker "github.com/kestfor/CrackHash/internal/services/worker/worker"
	"github.com/kestfor/CrackHash/internal/services/worker/worker/impl"
	"github.com/kestfor/CrackHash/internal/services/worker/workerservice"
	"github.com/kestfor/CrackHash/pkg/logging"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type workerFabric struct {
	id       uuid.UUID
	notifier notifier.Notifier
	cfg      *workerservice.Config
}

func newWorkerFabric(id uuid.UUID, n notifier.Notifier, cfg *workerservice.Config) *workerFabric {
	return &workerFabric{
		id:       id,
		notifier: n,
		cfg:      cfg,
	}
}

func (w *workerFabric) NewWorker() worker.Worker {
	return impl.NewWorker(w.id, []notifier.Notifier{w.notifier}, w.cfg.NotifyPeriod)
}

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

	initLogger(cfg.Logger, id)
	slog.Info("registered in manager")
	slog.Info("initializing dependencies...")

	httpNotifier := notifier.NewHTTPNotifier(cfg.Notifier)
	wrkFabric := newWorkerFabric(id, httpNotifier, cfg.Worker)

	workerService := workerservice.NewService(cfg.Worker, wrkFabric)
	slog.Info("dependencies initialized")

	initServer(cfg.HTTP, workerService)

	return nil
}

func initLogger(cfg *logging.LoggerConfig, id uuid.UUID) {
	logging.InitLogger(cfg,
		slog.Any("worker_id", id),
		slog.String("service", "worker"))
}

func initServer(httpServerConfig *HTTPServerConfig, workerService workersrv.Service) {
	slog.Info("initializing http server...")

	workerHandler := handler.NewHandler(workerService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/tasks/", workerHandler.HandleCreateTask)
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
