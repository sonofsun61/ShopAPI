package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
	"github.com/sonofsun61/APIFromSpec/internal/repository/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createSupplier(t *testing.T) uuid.UUID {
	t.Helper()
	repo := postgres.NewPostgresSupplierRepository(pool)
	id, err := repo.AddSupplier(context.Background(),
		entity.NewSupplierData{SupplierName: "Supplier", PhoneNumber: "+79999999999"},
		"Russia", "Moscow", "Arbat")
	require.NoError(t, err)
	return id
}

func TestSupplierRepository_AddAndGetByID(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresSupplierRepository(pool)

	id := createSupplier(t)

	got, err := repo.GetSupplierByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, entity.SupplierWithAddress{
		ID:           id,
		SupplierName: "Supplier",
		PhoneNumber:  "+79999999999",
		Country:      "Russia",
		City:         "Moscow",
		Street:       "Arbat",
	}, got)
}

func TestSupplierRepository_GetByID_NotFound(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresSupplierRepository(pool)

	_, err := repo.GetSupplierByID(context.Background(), uuid.New())
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestSupplierRepository_GetSuppliers(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresSupplierRepository(pool)
	for i := 0; i < 3; i++ {
		createSupplier(t)
	}
	limit := 2

	all, err := repo.GetSuppliers(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Len(t, all, 3)

	page, err := repo.GetSuppliers(context.Background(), &limit, nil)
	require.NoError(t, err)
	assert.Len(t, page, 2)
}

func TestSupplierRepository_UpdateSupplierAddress(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresSupplierRepository(pool)
	ctx := context.Background()
	id := createSupplier(t)

	require.NoError(t, repo.UpdateSupplierAddress(ctx, id, "Belarus", "Minsk", "Nezavisimosti"))

	got, err := repo.GetSupplierByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "Belarus", got.Country)
	assert.Equal(t, "Minsk", got.City)
	assert.Equal(t, "Nezavisimosti", got.Street)

	err = repo.UpdateSupplierAddress(ctx, uuid.New(), "Belarus", "Minsk", "Nezavisimosti")
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestSupplierRepository_DeleteSupplier(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresSupplierRepository(pool)
	ctx := context.Background()
	id := createSupplier(t)

	require.NoError(t, repo.DeleteSupplier(ctx, id))

	_, err := repo.GetSupplierByID(ctx, id)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	assert.ErrorIs(t, repo.DeleteSupplier(ctx, id), pgx.ErrNoRows)
}
