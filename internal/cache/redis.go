package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
	"github.com/sonofsun61/APIFromSpec/internal/service"
)

type CachedProductRepository struct {
	postgresRepo service.ProductRepository
	redisClient  *redis.Client
	ttl          time.Duration
	logger       *slog.Logger
}

func NewCachedProductRepository(postgresRepo service.ProductRepository, redisClient *redis.Client, ttl time.Duration, logger *slog.Logger) *CachedProductRepository {
	return &CachedProductRepository{
		postgresRepo: postgresRepo,
		redisClient:  redisClient,
		ttl:          ttl,
		logger:       logger,
	}
}

var _ service.ProductRepository = (*CachedProductRepository)(nil)

func (r *CachedProductRepository) CreateProduct(ctx context.Context, newProductData entity.NewProductData) (entity.Product, error) {
	return r.postgresRepo.CreateProduct(ctx, newProductData)
}

func (r *CachedProductRepository) DecreaseStock(ctx context.Context, productID uuid.UUID, amount int) error {
	if err := r.postgresRepo.DecreaseStock(ctx, productID, amount); err != nil {
		return err
	}
	r.invalidate(ctx, productID)
	return nil
}

func (r *CachedProductRepository) GetProductByID(ctx context.Context, productID uuid.UUID) (entity.Product, error) {
	key := productKey(productID)
	data, err := r.redisClient.Get(ctx, key).Bytes()
	if err == nil {
		var cached entity.Product
		if err := json.Unmarshal(data, &cached); err == nil {
			return cached, nil
		} else {
			r.logger.Warn("failed to decode cached product", "key", key, "err", err)
		}
	} else if !errors.Is(err, redis.Nil) {
		r.logger.Warn("redis get failed", "key", key, "err", err)
	}
	product, err := r.postgresRepo.GetProductByID(ctx, productID)
	if err != nil {
		return entity.Product{}, err
	}
	encoded, err := json.Marshal(product)
	if err != nil {
		r.logger.Warn("failed to encode product for cache", "key", key, "err", err)
		return product, nil
	}
	if err := r.redisClient.Set(ctx, key, encoded, r.ttl).Err(); err != nil {
		r.logger.Warn("redis set failed", "key", key, "err", err)
	}
	return product, nil
}

func (r *CachedProductRepository) GetAllProducts(ctx context.Context, limit *int, offset *int) ([]entity.Product, error) {
	return r.postgresRepo.GetAllProducts(ctx, limit, offset)
}

func (r *CachedProductRepository) DeleteProductByID(ctx context.Context, productID uuid.UUID) error {
	if err := r.postgresRepo.DeleteProductByID(ctx, productID); err != nil {
		return err
	}
	r.invalidate(ctx, productID)
	return nil
}

func (r *CachedProductRepository) invalidate(ctx context.Context, productID uuid.UUID) {
	key := productKey(productID)
	if err := r.redisClient.Del(ctx, key).Err(); err != nil {
		r.logger.Warn("redis del failed", "key", key, "err", err)
	}
}

func productKey(productID uuid.UUID) string {
	return "product:" + productID.String()
}
