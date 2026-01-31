package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	matchingv1 "github.com/linkflow/engine/api/gen/linkflow/matching/v1"
	"github.com/linkflow/engine/internal/matching"
	"github.com/linkflow/engine/internal/version"
)

func main() {
	var (
		port           = flag.Int("port", 7235, "gRPC server port")
		httpPort       = flag.Int("http-port", 8080, "HTTP server port")
		partitionCount = flag.Int("partition-count", 4, "Number of partitions")
	)
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	printBanner("Matching", logger)

	if *partitionCount < 1 || *partitionCount > math.MaxInt32 {
		logger.Error("invalid partition count", slog.Int("partition_count", *partitionCount))
		os.Exit(1)
	}

	svc := matching.NewService(matching.Config{
		NumPartitions: int32(*partitionCount),
		Replicas:      100,
		Logger:        logger,
	})

	server := grpc.NewServer()
	matchingv1.RegisterMatchingServiceServer(server, matching.NewGRPCServer(svc))
	reflection.Register(server)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		logger.Error("failed to listen", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("received signal, shutting down", slog.String("signal", sig.String()))
		cancel()
		server.GracefulStop()
	}()

	logger.Info("starting gRPC server", slog.Int("port", *port), slog.Int("partition_count", *partitionCount))

	go func() {
		if err := server.Serve(lis); err != nil {
			logger.Error("server failed", slog.String("error", err.Error()))
			cancel()
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

	<-ctx.Done()
	logger.Info("matching service stopped")
}

func printBanner(service string, logger *slog.Logger) {
	logger.Info(fmt.Sprintf("LinkFlow %s Service", service),
		slog.String("version", version.Version),
		slog.String("commit", version.GitCommit),
		slog.String("build_time", version.BuildTime),
	)
}
