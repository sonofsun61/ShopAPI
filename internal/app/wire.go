//go:build wireinject
// +build wireinject

package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sonofsun61/APIFromSpec/internal/authclient"
	"github.com/sonofsun61/APIFromSpec/internal/config"
	"github.com/sonofsun61/APIFromSpec/internal/database"
	"github.com/sonofsun61/APIFromSpec/internal/handler"
	"github.com/sonofsun61/APIFromSpec/internal/repository/postgres"
	"github.com/sonofsun61/APIFromSpec/internal/service"
	"google.golang.org/grpc"
)

func InitializeApp() *App {
	wire.Build(
		ProvideContext,
		ProvideValidatorOptions,
		ProvideConnString,
		validator.New,
		config.MustLoadConfig,
		database.MustConnectToDatabase,
		postgres.NewPostgresClientRepository,
		wire.Bind(new(service.ClientRepository), new(*postgres.PostgresClientRepository)),
		postgres.NewPostgresSupplierRepository,
		wire.Bind(new(service.SupplierRepository), new(*postgres.PostgresSupplierRepository)),
		postgres.NewPostgresProductRepository,
		wire.Bind(new(service.ProductRepository), new(*postgres.PostgresProductRepository)),
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
		handler.NewClientHandler,
		handler.NewSupplierHandler,
		handler.NewProductHandler,
		handler.NewImageHandler,
		handler.SetUpRouter,
		ProvideApp,
	)
	return nil
}

func ProvideApp(config *config.Config, pool *pgxpool.Pool, router http.Handler) *App {
	return &App{
		config: config,
		pool:   pool,
		router: router,
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

func ProvideAuthConn(cfg *config.Config) *grpc.ClientConn {
	grpcClientConn, err := authclient.NewConn(cfg.AuthServiceAddr)
	if err != nil {
		panic(fmt.Sprintf("could not create auth service connection: %v", err))
	}
	return grpcClientConn
}
