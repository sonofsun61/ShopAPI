package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
	"github.com/sonofsun61/APIFromSpec/internal/repository/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createProduct(t *testing.T, supplierID uuid.UUID, category string, stock int) entity.Product {
	t.Helper()
	repo := postgres.NewPostgresProductRepository(pool)
	product, err := repo.CreateProduct(context.Background(), entity.NewProductData{
		ProductName:    "Fridge",
		CategoryName:   category,
		Price:          decimal.RequireFromString("499.99"),
		AvailableStock: stock,
		SupplierID:     supplierID,
	})
	require.NoError(t, err)
	return product
}

func TestProductRepository_CreateAndGetByID(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresProductRepository(pool)
	supplierID := createSupplier(t)

	created := createProduct(t, supplierID, "Kitchen", 10)

	assert.NotEqual(t, uuid.Nil, created.ID)
	assert.NotEqual(t, uuid.Nil, created.CategoryID)
	assert.Equal(t, "Fridge", created.ProductName)
	assert.True(t, decimal.RequireFromString("499.99").Equal(created.Price))
	assert.Equal(t, 10, created.AvailableStock)
	assert.Equal(t, supplierID, created.SupplierID)
	assert.Nil(t, created.ImageID)

	got, err := repo.GetProductByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, created.CategoryID, got.CategoryID)
	assert.True(t, created.Price.Equal(got.Price))
}

func TestProductRepository_CreateProduct_ReusesCategory(t *testing.T) {
	cleanDB(t)
	supplierID := createSupplier(t)

	first := createProduct(t, supplierID, "Kitchen", 1)
	second := createProduct(t, supplierID, "Kitchen", 1)
	other := createProduct(t, supplierID, "Laundry", 1)

	assert.Equal(t, first.CategoryID, second.CategoryID, "same category name must map to one category")
	assert.NotEqual(t, first.CategoryID, other.CategoryID)
}

func TestProductRepository_CreateProduct_UnknownSupplier(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresProductRepository(pool)

	_, err := repo.CreateProduct(context.Background(), entity.NewProductData{
		ProductName:    "Fridge",
		CategoryName:   "Kitchen",
		Price:          decimal.NewFromInt(1),
		AvailableStock: 1,
		SupplierID:     uuid.New(),
	})
	require.Error(t, err)

	var categories int
	require.NoError(t, pool.QueryRow(context.Background(), "select count(*) from category").Scan(&categories))
	assert.Zero(t, categories)
}

func TestProductRepository_GetProductByID_NotFound(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresProductRepository(pool)

	_, err := repo.GetProductByID(context.Background(), uuid.New())
	require.ErrorIs(t, err, pgx.ErrNoRows, "the handler relies on pgx.ErrNoRows to answer 404")
}

func TestProductRepository_DecreaseStock(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresProductRepository(pool)
	ctx := context.Background()
	product := createProduct(t, createSupplier(t), "Kitchen", 5)

	require.NoError(t, repo.DecreaseStock(ctx, product.ID, 3))
	got, err := repo.GetProductByID(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.AvailableStock)

	err = repo.DecreaseStock(ctx, product.ID, 3)
	require.ErrorIs(t, err, pgx.ErrNoRows, "insufficient stock must be rejected")
	got, err = repo.GetProductByID(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.AvailableStock, "stock must stay unchanged after a rejected decrease")

	assert.ErrorIs(t, repo.DecreaseStock(ctx, uuid.New(), 1), pgx.ErrNoRows)
}

func TestProductRepository_GetAllProducts_OnlyInStock(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresProductRepository(pool)
	supplierID := createSupplier(t)
	inStock := createProduct(t, supplierID, "Kitchen", 3)
	createProduct(t, supplierID, "Kitchen", 0)

	products, err := repo.GetAllProducts(context.Background(), nil, nil)
	require.NoError(t, err)
	require.Len(t, products, 1)
	assert.Equal(t, inStock.ID, products[0].ID)
}

func TestProductRepository_DeleteProductByID(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresProductRepository(pool)
	ctx := context.Background()
	product := createProduct(t, createSupplier(t), "Kitchen", 1)

	require.NoError(t, repo.DeleteProductByID(ctx, product.ID))
	assert.ErrorIs(t, repo.DeleteProductByID(ctx, product.ID), pgx.ErrNoRows)
}
