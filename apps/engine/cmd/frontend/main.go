package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/linkflow/engine/internal/frontend"
	"github.com/linkflow/engine/internal/frontend/interceptor"
	"github.com/linkflow/engine/internal/version"
)

func main() {
	var (
		port         = flag.Int("port", 7233, "gRPC server port")
		httpPort     = flag.Int("http-port", 8080, "HTTP server port")
		historyAddr  = flag.String("history-addr", "localhost:7234", "History service address")
		matchingAddr = flag.String("matching-addr", "localhost:7235", "Matching service address")
	)
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	printBanner("Frontend", logger)

	_ = *historyAddr
	_ = *matchingAddr

	loggingInterceptor := interceptor.NewLoggingInterceptor(logger)
	authInterceptor := interceptor.NewAuthInterceptor(interceptor.AuthConfig{
		SkipMethods: []string{"/grpc.health.v1.Health/Check"},
	})

	svc := frontend.NewService(nil, nil, logger, frontend.DefaultServiceConfig())
	_ = svc

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			loggingInterceptor.UnaryInterceptor,
			authInterceptor.UnaryInterceptor,
		),
		grpc.ChainStreamInterceptor(
			loggingInterceptor.StreamInterceptor,
			authInterceptor.StreamInterceptor,
		),
	)

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

	logger.Info("starting gRPC server", slog.Int("port", *port))

	go func() {
		if err := server.Serve(lis); err != nil {
			logger.Error("server failed", slog.String("error", err.Error()))
			cancel()
		}
	}()

	// Start HTTP Server for Health Checks
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		httpServer := &http.Server{
			Addr:    fmt.Sprintf(":%d", *httpPort),
			Handler: mux,
		}

		logger.Info("starting HTTP server", slog.Int("port", *httpPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server failed", slog.String("error", err.Error()))
			cancel()
		}
	}()

	<-ctx.Done()
	logger.Info("frontend service stopped")
}

func printBanner(service string, logger *slog.Logger) {
	logger.Info(fmt.Sprintf("LinkFlow %s Service", service),
		slog.String("version", version.Version),
		slog.String("commit", version.GitCommit),
		slog.String("build_time", version.BuildTime),
	)
}
