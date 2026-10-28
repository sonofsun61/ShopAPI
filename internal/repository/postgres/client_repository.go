package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
)

type PostgresClientRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresClientRepository(pool *pgxpool.Pool) *PostgresClientRepository {
	return &PostgresClientRepository{
		pool: pool,
	}
}

func (r *PostgresClientRepository) CreateClient(ctx context.Context, newClientData entity.NewClientData, country string, city string, street string) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not open transaction to create client: %v", err)
	}
	defer tx.Rollback(ctx)
	var newAddressID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO address(country, city, street)
							VALUES ($1, $2, $3)
							RETURNING id`,
		country, city, street).Scan(&newAddressID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not scan address id into variable: %v", err)
	}
	var newClientID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO client(client_name, client_surname, birthday,
												gender, address_id)
							VALUES ($1, $2, $3, $4, $5)
							RETURNING id`,
		newClientData.ClientName, newClientData.ClientSurname, newClientData.Birthday, newClientData.Gender, newAddressID).Scan(&newClientID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not scan client id into variable: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("could not commit transaction: %v", err)
	}
	return newClientID, nil
}

func (r *PostgresClientRepository) DeleteClientByID(ctx context.Context, clientID uuid.UUID) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM client WHERE id = $1", clientID)
	if err != nil {
		return fmt.Errorf("could not delete client by id: %v", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresClientRepository) GetClientByNameAndSurname(ctx context.Context, name string, surname string) ([]entity.ClientWithAddress, error) {
	query := `
			SELECT 
				client.id, 
				client.client_name,
				client.client_surname,
				client.birthday,
				client.gender,
				client.registration_date, 
				address.country,
				address.city,
				address.street 
			FROM client
			JOIN address ON client.address_id = address.id
			WHERE client_name = $1 AND client_surname = $2`

	rows, err := r.pool.Query(ctx, query, name, surname)
	if err != nil {
		return []entity.ClientWithAddress{}, fmt.Errorf("could not select client by name, surname: %v", err)
	}
	clients, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.ClientWithAddress])
	if err != nil {
		return []entity.ClientWithAddress{}, fmt.Errorf("could not place row to struct: %v", err)
	}
	return clients, nil
}

func (r *PostgresClientRepository) GetAllClients(ctx context.Context, limit *int, offset *int) ([]entity.ClientWithAddress, error) {
	query := `
			SELECT 
				client.id, 
				client.client_name,
				client.client_surname,
				client.birthday,
				client.gender,
				client.registration_date, 
				address.country,
				address.city,
				address.street 
			FROM client
			JOIN address ON client.address_id = address.id
			LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return []entity.ClientWithAddress{}, fmt.Errorf("could not select clients: %v", err)
	}
	clients, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.ClientWithAddress])
	if err != nil {
		return []entity.ClientWithAddress{}, fmt.Errorf("could not place row to struct: %v", err)
	}
	return clients, nil
}

func (r *PostgresClientRepository) UpdateClientAddress(ctx context.Context, clientID uuid.UUID, country string, city string, street string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("could not open transaction to update client: %v", err)
	}
	defer tx.Rollback(ctx)
	var newAddressID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO address (country, city, street)
							VALUES ($1, $2, $3)
							RETURNING id`, country, city, street).Scan(&newAddressID)
	if err != nil {
		return fmt.Errorf("could not insert new address: %v", err)
	}
	tag, err := tx.Exec(ctx, "UPDATE client SET address_id = $1 WHERE id = $2", newAddressID, clientID)
	if err != nil {
		return fmt.Errorf("could not update client address: %v", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("could not commit transaction: %v", err)
	}
	return nil
}
