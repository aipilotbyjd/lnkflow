package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/linkflow/engine/internal/version"
	"github.com/linkflow/engine/internal/worker"
	"github.com/linkflow/engine/internal/worker/adapter"
	"github.com/linkflow/engine/internal/worker/executor"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		httpPort  = flag.Int("http-port", 8080, "HTTP server port")
		taskQueue = flag.String("task-queue", getEnv("TASK_QUEUE", "default"), "Task queue name")

		matchingAddr = flag.String("matching-addr", getEnv("MATCHING_ADDR", "localhost:7235"), "Matching service address")
		historyAddr  = flag.String("history-addr", getEnv("HISTORY_ADDR", "localhost:7234"), "History service address")
		numWorkers   = flag.Int("num-workers", 4, "Number of worker goroutines")
	)
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	printBanner("Worker", logger)

	// Connect to History Service
	historyConn, err := grpc.NewClient(*historyAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("failed to connect to history service", slog.String("error", err.Error()))
		os.Exit(1)
	}
	historyClient := adapter.NewHistoryClient(historyConn)

	svc, err := worker.NewService(worker.Config{
		TaskQueues:    strings.Split(*taskQueue, ","),
		Identity:      fmt.Sprintf("worker-%d", os.Getpid()),
		MatchingAddr:  *matchingAddr,
		PollInterval:  time.Second,
		Logger:        logger,
		HistoryClient: historyClient,
	})
	if err != nil {
		return fmt.Errorf("failed to create worker service: %w", err)
	}

	// Register Workflow Executor
	workflowExecutor := executor.NewWorkflowExecutor(historyClient, logger)
	svc.RegisterExecutor(workflowExecutor)

	httpExecutor := executor.NewHTTPExecutor()
	svc.RegisterExecutor(httpExecutor)

	// Register additional executors
	transformExecutor := executor.NewTransformExecutor()
	svc.RegisterExecutor(transformExecutor)

	loopExecutor := executor.NewLoopExecutor()
	svc.RegisterExecutor(loopExecutor)

	conditionExecutor := executor.NewConditionExecutor()
	svc.RegisterExecutor(conditionExecutor)

	emailExecutor := executor.NewEmailExecutor()
	svc.RegisterExecutor(emailExecutor)

	delayExecutor := executor.NewDelayExecutor()
	svc.RegisterExecutor(delayExecutor)

	aiExecutor := executor.NewAIExecutor()
	svc.RegisterExecutor(aiExecutor)

	webhookExecutor := executor.NewWebhookExecutor()
	svc.RegisterExecutor(webhookExecutor)

	manualExecutor := executor.NewManualExecutor()
	svc.RegisterExecutor(manualExecutor)

	slackExecutor := executor.NewSlackExecutor()
	svc.RegisterExecutor(slackExecutor)

	discordExecutor := executor.NewDiscordExecutor()
	svc.RegisterExecutor(discordExecutor)

	twilioExecutor := executor.NewTwilioExecutor()
	svc.RegisterExecutor(twilioExecutor)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := svc.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker service: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("received signal, shutting down", slog.String("signal", sig.String()))
		cancel()
		if err := svc.Stop(); err != nil {
			logger.Error("failed to stop worker service", slog.String("error", err.Error()))
		}
	}()

	// Start HTTP Server for Health Checks
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		})

		httpServer := &http.Server{
			Addr:              fmt.Sprintf(":%d", *httpPort),
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       120 * time.Second,
		}

		logger.Info("starting HTTP server", slog.Int("port", *httpPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server failed", slog.String("error", err.Error()))
			cancel()
		}
	}()

	logger.Info("worker pool started",
		slog.String("task_queue", *taskQueue),
		slog.String("matching_addr", *matchingAddr),
		slog.Int("num_workers", *numWorkers),
	)

	<-ctx.Done()
	logger.Info("worker service stopped")
	return nil
}

func printBanner(service string, logger *slog.Logger) {
	logger.Info(fmt.Sprintf("LinkFlow %s Service", service),
		slog.String("version", version.Version),
		slog.String("commit", version.GitCommit),
		slog.String("build_time", version.BuildTime),
	)
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
