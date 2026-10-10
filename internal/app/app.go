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
	"github.com/sonofsun61/APIFromSpec/internal/config"
)

type App struct {
	config *config.Config
	pool   *pgxpool.Pool
	router http.Handler
	logger *slog.Logger
}

func (a *App) Run() error {
	defer a.pool.Close()
	server := &http.Server{
		Addr:    ":8080",
		Handler: a.router,
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
