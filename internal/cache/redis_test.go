package cache_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/sonofsun61/APIFromSpec/internal/cache"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
	"github.com/sonofsun61/APIFromSpec/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testTTL = 5 * time.Minute

var (
	discardLogger = slog.New(slog.DiscardHandler)
	errDBDown     = errors.New("db down")
)

type mockProductRepo struct {
	mock.Mock
}

var _ service.ProductRepository = (*mockProductRepo)(nil)

func (m *mockProductRepo) CreateProduct(ctx context.Context, newProductData entity.NewProductData) (entity.Product, error) {
	args := m.Called(ctx, newProductData)
	return args.Get(0).(entity.Product), args.Error(1)
}

func (m *mockProductRepo) DecreaseStock(ctx context.Context, productID uuid.UUID, amount int) error {
	args := m.Called(ctx, productID, amount)
	return args.Error(0)
}

func (m *mockProductRepo) GetProductByID(ctx context.Context, productID uuid.UUID) (entity.Product, error) {
	args := m.Called(ctx, productID)
	return args.Get(0).(entity.Product), args.Error(1)
}

func (m *mockProductRepo) GetAllProducts(ctx context.Context, limit *int, offset *int) ([]entity.Product, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]entity.Product), args.Error(1)
}

func (m *mockProductRepo) DeleteProductByID(ctx context.Context, productID uuid.UUID) error {
	args := m.Called(ctx, productID)
	return args.Error(0)
}

type testEnv struct {
	repo   *mockProductRepo
	redis  *miniredis.Miniredis
	cached *cache.CachedProductRepository
}

func newTestEnv(t *testing.T) testEnv {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	repo := new(mockProductRepo)
	return testEnv{
		repo:   repo,
		redis:  mr,
		cached: cache.NewCachedProductRepository(repo, client, testTTL, discardLogger),
	}
}

func newProduct() entity.Product {
	imageID := uuid.New()
	return entity.Product{
		ID:             uuid.New(),
		ProductName:    "Fridge",
		CategoryID:     uuid.New(),
		Price:          decimal.RequireFromString("499.99"),
		AvailableStock: 10,
		LastUpdateDate: time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
		SupplierID:     uuid.New(),
		ImageID:        &imageID,
	}
}

func key(id uuid.UUID) string {
	return "product:" + id.String()
}

func assertSameProduct(t *testing.T, want, got entity.Product) {
	t.Helper()
	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.ProductName, got.ProductName)
	assert.Equal(t, want.CategoryID, got.CategoryID)
	assert.True(t, want.Price.Equal(got.Price), "price: want %s, got %s", want.Price, got.Price)
	assert.Equal(t, want.AvailableStock, got.AvailableStock)
	assert.True(t, want.LastUpdateDate.Equal(got.LastUpdateDate))
	assert.Equal(t, want.SupplierID, got.SupplierID)
	assert.Equal(t, want.ImageID, got.ImageID)
}

func TestGetProductByID_MissLoadsFromRepoAndCaches(t *testing.T) {
	env := newTestEnv(t)
	product := newProduct()
	env.repo.On("GetProductByID", mock.Anything, product.ID).Return(product, nil).Once()

	got, err := env.cached.GetProductByID(context.Background(), product.ID)

	require.NoError(t, err)
	assertSameProduct(t, product, got)
	require.True(t, env.redis.Exists(key(product.ID)), "product must be stored in redis after a miss")
	assert.Equal(t, testTTL, env.redis.TTL(key(product.ID)))
	env.repo.AssertExpectations(t)
}

func TestGetProductByID_HitDoesNotCallRepo(t *testing.T) {
	env := newTestEnv(t)
	product := newProduct()
	env.repo.On("GetProductByID", mock.Anything, product.ID).Return(product, nil).Once()
	ctx := context.Background()

	_, err := env.cached.GetProductByID(ctx, product.ID)
	require.NoError(t, err)
	got, err := env.cached.GetProductByID(ctx, product.ID)

	require.NoError(t, err)
	assertSameProduct(t, product, got)
	env.repo.AssertNumberOfCalls(t, "GetProductByID", 1)
}

func TestGetProductByID_HitFromPrefilledCache(t *testing.T) {
	env := newTestEnv(t)
	product := newProduct()
	encoded, err := json.Marshal(product)
	require.NoError(t, err)
	require.NoError(t, env.redis.Set(key(product.ID), string(encoded)))

	got, err := env.cached.GetProductByID(context.Background(), product.ID)

	require.NoError(t, err)
	assertSameProduct(t, product, got)
	env.repo.AssertNotCalled(t, "GetProductByID", mock.Anything, mock.Anything)
}

func TestGetProductByID_ExpiredEntryIsReloaded(t *testing.T) {
	env := newTestEnv(t)
	product := newProduct()
	env.repo.On("GetProductByID", mock.Anything, product.ID).Return(product, nil).Twice()
	ctx := context.Background()

	_, err := env.cached.GetProductByID(ctx, product.ID)
	require.NoError(t, err)
	env.redis.FastForward(testTTL + time.Second)
	_, err = env.cached.GetProductByID(ctx, product.ID)

	require.NoError(t, err)
	env.repo.AssertNumberOfCalls(t, "GetProductByID", 2)
}

func TestGetProductByID_CorruptedEntryIsReplaced(t *testing.T) {
	env := newTestEnv(t)
	product := newProduct()
	require.NoError(t, env.redis.Set(key(product.ID), "not json"))
	env.repo.On("GetProductByID", mock.Anything, product.ID).Return(product, nil).Once()

	got, err := env.cached.GetProductByID(context.Background(), product.ID)

	require.NoError(t, err)
	assertSameProduct(t, product, got)
	stored, err := env.redis.Get(key(product.ID))
	require.NoError(t, err)
	var decoded entity.Product
	require.NoError(t, json.Unmarshal([]byte(stored), &decoded), "corrupted entry must be overwritten with valid JSON")
	assertSameProduct(t, product, decoded)
}

func TestGetProductByID_RepoErrorIsNotCached(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"not found", pgx.ErrNoRows},
		{"db error", errDBDown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			id := uuid.New()
			env.repo.On("GetProductByID", mock.Anything, id).Return(entity.Product{}, tt.err).Once()

			_, err := env.cached.GetProductByID(context.Background(), id)

			require.ErrorIs(t, err, tt.err)
			assert.False(t, env.redis.Exists(key(id)))
		})
	}
}

func TestGetProductByID_RedisDownFallsBackToRepo(t *testing.T) {
	env := newTestEnv(t)
	product := newProduct()
	env.repo.On("GetProductByID", mock.Anything, product.ID).Return(product, nil)
	env.redis.Close()

	got, err := env.cached.GetProductByID(context.Background(), product.ID)

	require.NoError(t, err)
	assertSameProduct(t, product, got)
	env.repo.AssertExpectations(t)
}

func TestInvalidation(t *testing.T) {
	ops := []struct {
		name   string
		method string
		args   func(id uuid.UUID) []any
		call   func(c *cache.CachedProductRepository, id uuid.UUID) error
	}{
		{
			name:   "DecreaseStock",
			method: "DecreaseStock",
			args:   func(id uuid.UUID) []any { return []any{mock.Anything, id, 3} },
			call: func(c *cache.CachedProductRepository, id uuid.UUID) error {
				return c.DecreaseStock(context.Background(), id, 3)
			},
		},
		{
			name:   "DeleteProductByID",
			method: "DeleteProductByID",
			args:   func(id uuid.UUID) []any { return []any{mock.Anything, id} },
			call: func(c *cache.CachedProductRepository, id uuid.UUID) error {
				return c.DeleteProductByID(context.Background(), id)
			},
		},
	}
	for _, op := range ops {
		t.Run(op.name+"/success removes cached product", func(t *testing.T) {
			env := newTestEnv(t)
			id := uuid.New()
			require.NoError(t, env.redis.Set(key(id), "{}"))
			env.repo.On(op.method, op.args(id)...).Return(nil).Once()

			require.NoError(t, op.call(env.cached, id))

			assert.False(t, env.redis.Exists(key(id)))
			env.repo.AssertExpectations(t)
		})

		t.Run(op.name+"/repo error keeps cached product", func(t *testing.T) {
			env := newTestEnv(t)
			id := uuid.New()
			require.NoError(t, env.redis.Set(key(id), "{}"))
			env.repo.On(op.method, op.args(id)...).Return(pgx.ErrNoRows).Once()

			err := op.call(env.cached, id)

			require.ErrorIs(t, err, pgx.ErrNoRows)
			assert.True(t, env.redis.Exists(key(id)))
		})

		t.Run(op.name+"/redis down does not fail the write", func(t *testing.T) {
			env := newTestEnv(t)
			id := uuid.New()
			env.repo.On(op.method, op.args(id)...).Return(nil).Once()
			env.redis.Close()

			require.NoError(t, op.call(env.cached, id))
			env.repo.AssertExpectations(t)
		})
	}
}

func TestDecreaseStock_NextReadReturnsFreshData(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	before := newProduct()
	after := before
	after.AvailableStock = 7
	env.repo.On("GetProductByID", mock.Anything, before.ID).Return(before, nil).Once()
	env.repo.On("DecreaseStock", mock.Anything, before.ID, 3).Return(nil).Once()
	env.repo.On("GetProductByID", mock.Anything, before.ID).Return(after, nil).Once()

	_, err := env.cached.GetProductByID(ctx, before.ID)
	require.NoError(t, err)
	require.NoError(t, env.cached.DecreaseStock(ctx, before.ID, 3))
	got, err := env.cached.GetProductByID(ctx, before.ID)

	require.NoError(t, err)
	assert.Equal(t, 7, got.AvailableStock)
	env.repo.AssertExpectations(t)
}

func TestPassThroughMethods(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	product := newProduct()
	data := entity.NewProductData{ProductName: "Fridge", CategoryName: "Kitchen", AvailableStock: 1}
	limit := 10
	env.repo.On("CreateProduct", mock.Anything, data).Return(product, nil).Once()
	env.repo.On("GetAllProducts", mock.Anything, &limit, (*int)(nil)).Return([]entity.Product{product}, nil).Once()

	created, err := env.cached.CreateProduct(ctx, data)
	require.NoError(t, err)
	assert.Equal(t, product.ID, created.ID)

	all, err := env.cached.GetAllProducts(ctx, &limit, nil)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	assert.Empty(t, env.redis.Keys(), "pass-through methods must not touch the cache")
	env.repo.AssertExpectations(t)
}
