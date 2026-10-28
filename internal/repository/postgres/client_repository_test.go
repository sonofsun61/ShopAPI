package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
	"github.com/sonofsun61/APIFromSpec/internal/repository/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newClientData(name, surname string) entity.NewClientData {
	return entity.NewClientData{
		ClientName:    name,
		ClientSurname: surname,
		Birthday:      time.Date(1990, time.May, 1, 0, 0, 0, 0, time.UTC),
		Gender:        "male",
	}
}

func TestClientRepository_CreateAndGetByName(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresClientRepository(pool)
	ctx := context.Background()
	data := newClientData("Ivan", "Ivanov")

	id, err := repo.CreateClient(ctx, data, "Russia", "Moscow", "Tverskaya")
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id)
	_, err = repo.CreateClient(ctx, newClientData("Petr", "Petrov"), "Russia", "Kazan", "Baumana")
	require.NoError(t, err)

	clients, err := repo.GetClientByNameAndSurname(ctx, "Ivan", "Ivanov")
	require.NoError(t, err)
	require.Len(t, clients, 1)
	got := clients[0]
	assert.Equal(t, id, got.ID)
	assert.Equal(t, data.ClientName, got.ClientName)
	assert.Equal(t, data.ClientSurname, got.ClientSurname)
	assert.True(t, data.Birthday.Equal(got.Birthday))
	assert.Equal(t, data.Gender, got.Gender)
	assert.Equal(t, "Russia", got.Country)
	assert.Equal(t, "Moscow", got.City)
	assert.Equal(t, "Tverskaya", got.Street)
}

func TestClientRepository_GetByName_NoMatches(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresClientRepository(pool)

	clients, err := repo.GetClientByNameAndSurname(context.Background(), "Nobody", "Nobody")
	require.NoError(t, err)
	assert.Empty(t, clients)
}

func TestClientRepository_CreateClient_InvalidGender(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresClientRepository(pool)
	ctx := context.Background()
	data := newClientData("Ivan", "Ivanov")
	data.Gender = "unknown"

	_, err := repo.CreateClient(ctx, data, "Russia", "Moscow", "Tverskaya")
	require.Error(t, err)

	var addresses int
	require.NoError(t, pool.QueryRow(ctx, "select count(*) from address").Scan(&addresses))
	assert.Zero(t, addresses)
}

func TestClientRepository_GetAllClients_Pagination(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresClientRepository(pool)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := repo.CreateClient(ctx, newClientData("Ivan", "Ivanov"), "Russia", "Moscow", "Tverskaya")
		require.NoError(t, err)
	}
	two, one := 2, 1

	tests := []struct {
		name   string
		limit  *int
		offset *int
		want   int
	}{
		{"no limit", nil, nil, 3},
		{"limit", &two, nil, 2},
		{"offset", nil, &one, 2},
		{"limit and offset", &two, &two, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clients, err := repo.GetAllClients(ctx, tt.limit, tt.offset)
			require.NoError(t, err)
			assert.Len(t, clients, tt.want)
		})
	}
}

func TestClientRepository_UpdateClientAddress(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresClientRepository(pool)
	ctx := context.Background()
	id, err := repo.CreateClient(ctx, newClientData("Ivan", "Ivanov"), "Russia", "Moscow", "Tverskaya")
	require.NoError(t, err)

	require.NoError(t, repo.UpdateClientAddress(ctx, id, "Russia", "Sochi", "Kurortny"))

	clients, err := repo.GetClientByNameAndSurname(ctx, "Ivan", "Ivanov")
	require.NoError(t, err)
	require.Len(t, clients, 1)
	assert.Equal(t, "Sochi", clients[0].City)
	assert.Equal(t, "Kurortny", clients[0].Street)
}

func TestClientRepository_UpdateClientAddress_NotFound(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresClientRepository(pool)
	ctx := context.Background()

	err := repo.UpdateClientAddress(ctx, uuid.New(), "Russia", "Sochi", "Kurortny")
	require.ErrorIs(t, err, pgx.ErrNoRows)

	var addresses int
	require.NoError(t, pool.QueryRow(ctx, "select count(*) from address").Scan(&addresses))
	assert.Zero(t, addresses, "new address must be rolled back when the client does not exist")
}

func TestClientRepository_DeleteClientByID(t *testing.T) {
	cleanDB(t)
	repo := postgres.NewPostgresClientRepository(pool)
	ctx := context.Background()
	id, err := repo.CreateClient(ctx, newClientData("Ivan", "Ivanov"), "Russia", "Moscow", "Tverskaya")
	require.NoError(t, err)

	require.NoError(t, repo.DeleteClientByID(ctx, id))

	clients, err := repo.GetClientByNameAndSurname(ctx, "Ivan", "Ivanov")
	require.NoError(t, err)
	assert.Empty(t, clients)

	err = repo.DeleteClientByID(ctx, id)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "deleting twice must report not found")
}
