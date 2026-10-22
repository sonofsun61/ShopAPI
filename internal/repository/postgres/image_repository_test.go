package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sonofsun61/APIFromSpec/internal/repository/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	pngBytes  = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	jpegBytes = []byte{0xff, 0xd8, 0xff, 0xe0}
)

func TestImageRepository_AddAndGet(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresImageRepository(pool)
	ctx := context.Background()
	product := createProduct(t, createSupplier(t), "Kitchen", 1)

	require.NoError(t, repo.AddImage(ctx, pngBytes, product.ID))

	byProduct, err := repo.GetImageByProductID(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, pngBytes, byProduct.Image)
	require.NotEqual(t, uuid.Nil, byProduct.ID)

	byID, err := repo.GetImageByImageID(ctx, byProduct.ID)
	require.NoError(t, err)
	assert.Equal(t, byProduct, byID)
}

func TestImageRepository_AddImage_ProductNotFound(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresImageRepository(pool)
	ctx := context.Background()

	err := repo.AddImage(ctx, pngBytes, uuid.New())
	require.ErrorIs(t, err, pgx.ErrNoRows)

	var images int
	require.NoError(t, pool.QueryRow(ctx, "select count(*) from images").Scan(&images))
	assert.Zero(t, images, "image insert must be rolled back when the product does not exist")
}

func TestImageRepository_ChangeImage(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresImageRepository(pool)
	ctx := context.Background()
	product := createProduct(t, createSupplier(t), "Kitchen", 1)
	require.NoError(t, repo.AddImage(ctx, pngBytes, product.ID))
	image, err := repo.GetImageByProductID(ctx, product.ID)
	require.NoError(t, err)

	require.NoError(t, repo.ChangeImage(ctx, image.ID, jpegBytes))

	got, err := repo.GetImageByImageID(ctx, image.ID)
	require.NoError(t, err)
	assert.Equal(t, jpegBytes, got.Image)
	assert.ErrorIs(t, repo.ChangeImage(ctx, uuid.New(), jpegBytes), pgx.ErrNoRows)
}

func TestImageRepository_DeleteImage(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresImageRepository(pool)
	productRepo := postgres.NewPostgresProductRepository(pool)
	ctx := context.Background()
	product := createProduct(t, createSupplier(t), "Kitchen", 1)
	require.NoError(t, repo.AddImage(ctx, pngBytes, product.ID))
	image, err := repo.GetImageByProductID(ctx, product.ID)
	require.NoError(t, err)

	require.NoError(t, repo.DeleteImage(ctx, image.ID))

	got, err := productRepo.GetProductByID(ctx, product.ID)
	require.NoError(t, err)
	assert.Nil(t, got.ImageID, "product.image_id must be set to null (on delete set null)")
	_, err = repo.GetImageByProductID(ctx, product.ID)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
	assert.ErrorIs(t, repo.DeleteImage(ctx, image.ID), pgx.ErrNoRows)
}

func TestImageRepository_GetImage_NotFound(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresImageRepository(pool)
	ctx := context.Background()

	_, err := repo.GetImageByProductID(ctx, uuid.New())
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	_, err = repo.GetImageByImageID(ctx, uuid.New())
	assert.ErrorIs(t, err, pgx.ErrNoRows, "the handler relies on pgx.ErrNoRows to answer 404")
}
