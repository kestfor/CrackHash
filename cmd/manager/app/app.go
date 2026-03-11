package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/kestfor/CrackHash/cmd/manager/handler"
	"github.com/kestfor/CrackHash/internal/services/broker/rabbitmq"
	"github.com/kestfor/CrackHash/internal/services/manager"
	"github.com/kestfor/CrackHash/internal/services/manager/service"
	"github.com/kestfor/CrackHash/internal/services/manager/storage/mongodb"
	"github.com/kestfor/CrackHash/pkg/logging"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Storage.URL))
	if err != nil {
		return err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("db connection not healthy: %w", err)
	}

	db := client.Database(cfg.Storage.DB)

	progressStorage, err := mongodb.NewTaskProgressStorage(db)
	if err != nil {
		return fmt.Errorf("create task progress storage: %w", err)
	}

	subTaskStorage, err := mongodb.NewSubTaskStorage(db)
	if err != nil {
		return fmt.Errorf("create subtask storage: %w", err)
	}

	conn, err := amqp.Dial(cfg.Broker.URL)
	if err != nil {
		return err
	}

	if err := rabbitmq.DefineQueues(conn, cfg.Broker.RequeueLimit); err != nil {
		conn.Close()
		return err
	}
	conn.Close()

	progressConsumer := rabbitmq.NewConsumer(cfg.Broker.URL, rabbitmq.TasksProgressQueue, 0)
	deadLettersConsumer := rabbitmq.NewConsumer(cfg.Broker.URL, rabbitmq.DeadLetterQueue, 0)

	tasksPublisher, err := rabbitmq.NewPublisher(cfg.Broker.URL)
	if err != nil {
		return err
	}
	defer tasksPublisher.Close()

	managerService := service.NewService(cfg.HashCracker.Alphabet, progressStorage, subTaskStorage, tasksPublisher, progressConsumer, deadLettersConsumer, cfg.RetrySendPeriod)
	go func() {
		if err := managerService.Run(ctx); err != nil {
			slog.Error("manager service run failed", slog.Any("error", err))
		}
		slog.Info("manager service stopped")
	}()

	initServer(cfg.HTTP, managerService)

	return nil
}

func initServer(httpServerConfig *HTTPConfig, managerService manager.Service) {
	slog.Info("initializing http server...")

	workerHandler := handler.NewHandler(managerService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/hash/crack", workerHandler.HandleCreateTask)
	mux.HandleFunc("GET /api/hash/status", workerHandler.HandleGetTaskProgress)
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
