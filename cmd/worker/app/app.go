package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/broker"
	"github.com/kestfor/CrackHash/internal/services/broker/rabbitmq"
	worker "github.com/kestfor/CrackHash/internal/services/worker/worker"
	"github.com/kestfor/CrackHash/internal/services/worker/worker/impl"
	"github.com/kestfor/CrackHash/internal/services/worker/workerservice"
	"github.com/kestfor/CrackHash/pkg/logging"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type workerFabric struct {
	id        uuid.UUID
	publisher broker.Publisher
	cfg       *workerservice.Config
}

func newWorkerFabric(wID uuid.UUID, cfg *workerservice.Config, publisher broker.Publisher) *workerFabric {
	return &workerFabric{
		id:        wID,
		publisher: publisher,
		cfg:       cfg,
	}
}

func (w *workerFabric) NewWorker() worker.Worker {
	return impl.NewWorker(w.id, w.publisher, w.cfg.NotifyPeriod)
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
	workerID := uuid.New()
	initLogger(cfg.Logger, workerID)

	conn, err := amqp.Dial(cfg.Broker.URL)
	if err != nil {
		return err
	}

	if err = rabbitmq.DefineQueues(conn, cfg.Broker.RequeueLimit); err != nil {
		conn.Close()
		return err
	}
	conn.Close()

	progressPublisher, err := rabbitmq.NewPublisher(cfg.Broker.URL)
	if err != nil {
		return err
	}
	defer progressPublisher.Close()

	tasksConsumer := rabbitmq.NewConsumer(cfg.Broker.URL, rabbitmq.TasksQueue, cfg.Worker.MaxParallel)

	wrkFabric := newWorkerFabric(workerID, cfg.Worker, progressPublisher)

	workerService := workerservice.NewService(wrkFabric, tasksConsumer)
	slog.Info("dependencies initialized")

	go func() {
		err := workerService.Run(context.Background())
		if err != nil {
			slog.Error("worker service failed", slog.Any("error", err))
		}
		slog.Info("worker service stopped")
	}()

	initServer(cfg.HTTP)

	return nil
}

func initLogger(cfg *logging.LoggerConfig, id uuid.UUID) {
	logging.InitLogger(cfg,
		slog.Any("worker_id", id),
		slog.String("service", "worker"))
}

func initServer(httpServerConfig *HTTPServerConfig) {
	slog.Info("initializing http server...")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)

	slog.Info("http server initialized, serving...")
	err := http.ListenAndServe(fmt.Sprintf(":%d", httpServerConfig.Port), mux)
	if err != nil {
		slog.Error("failed to start server", "error", err)
	}

}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
