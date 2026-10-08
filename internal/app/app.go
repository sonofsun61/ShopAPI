package app

import (
	"context"
	"fmt"
	"log"
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
}

func (a *App) Run() error {
	server := &http.Server{
		Addr:    ":8080",
		Handler: a.router,
	}
	go func() {
		log.Println("server started")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	log.Println("Server is shutting down...")

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}
	a.pool.Close()
	log.Println("Server has been stopped")
	return nil
}
