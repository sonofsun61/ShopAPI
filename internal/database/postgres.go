package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConnString string

func MustConnectToDatabase(ctx context.Context, connString ConnString) *pgxpool.Pool {
	pool, err := pgxpool.New(ctx, string(connString))
	if err != nil {
		panic(fmt.Sprintf("could not create pgxpool: %v", err))
	}
	if err := pool.Ping(ctx); err != nil {
		panic(fmt.Sprintf("could not ping pgxpool: %v", err))
	}
	return pool
}
