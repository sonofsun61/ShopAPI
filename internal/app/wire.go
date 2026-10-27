//go:build wireinject
// +build wireinject

package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/sonofsun61/APIFromSpec/internal/authclient"
	"github.com/sonofsun61/APIFromSpec/internal/cache"
	"github.com/sonofsun61/APIFromSpec/internal/config"
	"github.com/sonofsun61/APIFromSpec/internal/database"
	"github.com/sonofsun61/APIFromSpec/internal/handler"
	"github.com/sonofsun61/APIFromSpec/internal/logger"
	"github.com/sonofsun61/APIFromSpec/internal/middleware"
	"github.com/sonofsun61/APIFromSpec/internal/repository/postgres"
	"github.com/sonofsun61/APIFromSpec/internal/service"
	"google.golang.org/grpc"
)

func InitializeApp() *App {
	wire.Build(
		ProvideContext,
		ProvideValidatorOptions,
		ProvideConnString,
		logger.NewLogger,
		validator.New,
		config.MustLoadConfig,
		database.MustConnectToDatabase,
		postgres.NewPostgresClientRepository,
		wire.Bind(new(service.ClientRepository), new(*postgres.PostgresClientRepository)),
		postgres.NewPostgresSupplierRepository,
		wire.Bind(new(service.SupplierRepository), new(*postgres.PostgresSupplierRepository)),
		postgres.NewPostgresProductRepository,
		ProvideRedisClient,
		ProvideCachedProductRepository,
		wire.Bind(new(service.ProductRepository), new(*cache.CachedProductRepository)),
		postgres.NewPostgresImageRepository,
		wire.Bind(new(service.ImageRepository), new(*postgres.PostgresImageRepository)),
		service.NewClientService,
		wire.Bind(new(handler.ClientService), new(*service.ClientService)),
		service.NewSupplierService,
		wire.Bind(new(handler.SupplierService), new(*service.SupplierService)),
		service.NewProductService,
		wire.Bind(new(handler.ProductService), new(*service.ProductService)),
		service.NewImageService,
		wire.Bind(new(handler.ImageService), new(*service.ImageService)),
		ProvideAuthConn,
		authclient.New,
		wire.Bind(new(handler.AuthService), new(*authclient.Client)),
		handler.NewAuthHandler,
		wire.Bind(new(middleware.TokenValidator), new(*authclient.Client)),
		handler.NewClientHandler,
		handler.NewSupplierHandler,
		handler.NewProductHandler,
		handler.NewImageHandler,
		handler.SetUpRouter,
		ProvideApp,
	)
	return nil
}

const productCacheTTL = 5 * time.Minute

func ProvideApp(config *config.Config, pool *pgxpool.Pool,
	router http.Handler, logger *slog.Logger,
	authConn *grpc.ClientConn, redisClient *redis.Client) *App {
	return &App{
		config:      config,
		pool:        pool,
		router:      router,
		logger:      logger,
		authConn:    authConn,
		redisClient: redisClient,
	}
}

func ProvideContext() context.Context {
	return context.Background()
}

func ProvideValidatorOptions() []validator.Option {
	return nil
}

func ProvideConnString(cfg *config.Config) database.ConnString {
	return database.ConnString(cfg.ConnString)
}

func ProvideRedisClient(ctx context.Context, cfg *config.Config, logger *slog.Logger) *redis.Client {
	client := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		logger.Warn("redis is unavailable, product cache is disabled until it comes back", "addr", cfg.RedisAddr, "err", err)
	}
	return client
}

func ProvideCachedProductRepository(repo *postgres.PostgresProductRepository, redisClient *redis.Client, logger *slog.Logger) *cache.CachedProductRepository {
	return cache.NewCachedProductRepository(repo, redisClient, productCacheTTL, logger)
}

func ProvideAuthConn(cfg *config.Config) *grpc.ClientConn {
	grpcClientConn, err := authclient.NewConn(cfg.AuthServiceAddr)
	if err != nil {
		panic(fmt.Sprintf("could not create auth service connection: %v", err))
	}
	return grpcClientConn
}
