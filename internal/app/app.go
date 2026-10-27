package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/sonofsun61/APIFromSpec/internal/config"
	"google.golang.org/grpc"
)

type App struct {
	config      *config.Config
	pool        *pgxpool.Pool
	router      http.Handler
	logger      *slog.Logger
	authConn    *grpc.ClientConn
	redisClient *redis.Client
}

func (a *App) Run() error {
	defer a.pool.Close()
	defer func() {
		if err := a.redisClient.Close(); err != nil {
			a.logger.Error("failed to close redis client", "err", err)
		}
	}()
	defer func() {
		if err := a.authConn.Close(); err != nil {
			a.logger.Error("failed to close auth connection", "err", err)
		}
	}()
	server := &http.Server{
		Addr:              ":8080",
		Handler:           a.router,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", server.Addr, err)
	}
	a.logger.Info("server started", "port", server.Addr)

	errCh := make(chan error, 1)
	go func() {
		server.Serve(listener)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		a.logger.Info("server is shutting down")

		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("server forced to shutdown: %w", err)
		}
		a.logger.Info("server has been stopped")
		return nil
	case err := <-errCh:
		return fmt.Errorf("server failed: %w", err)
	}
}
