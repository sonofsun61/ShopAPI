package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
)

type PostgresSupplierRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSupplierRepository(pool *pgxpool.Pool) *PostgresSupplierRepository {
	return &PostgresSupplierRepository{
		pool: pool,
	}
}

func (r *PostgresSupplierRepository) AddSupplier(ctx context.Context, newSupplierData entity.NewSupplierData, country string, city string, street string) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)
	var newAddressID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO address (country, city, street)
							VALUES ($1, $2, $3) RETURNING id`,
		country, city, street).Scan(&newAddressID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert new supplier address: %v", err)
	}
	var newSupplierID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO supplier (name, phone_number, address_id) VALUES ($1, $2, $3) RETURNING id`, newSupplierData.SupplierName, newSupplierData.PhoneNumber, newAddressID).Scan(&newSupplierID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert new supplier: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("could not commit transaction: %v", err)
	}
	return newSupplierID, nil
}

func (r *PostgresSupplierRepository) UpdateSupplierAddress(ctx context.Context, supplierID uuid.UUID, country string, city string, street string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)
	var newAddressID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO address (country, city, street)
							VALUES ($1, $2, $3) RETURNING id`,
		country, city, street).Scan(&newAddressID)
	if err != nil {
		return fmt.Errorf("failed to insert new address: %v", err)
	}
	tag, err := tx.Exec(ctx, "UPDATE supplier SET address_id = $1 WHERE id = $2", newAddressID, supplierID)
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

func (r *PostgresSupplierRepository) DeleteSupplier(ctx context.Context, supplierID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM supplier WHERE id = $1`, supplierID)
	if err != nil {
		return fmt.Errorf("could not delete supplier: %v", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresSupplierRepository) GetSuppliers(ctx context.Context, limit *int, offset *int) ([]entity.SupplierWithAddress, error) {
	query := `
			SELECT 
				supplier.id, 
				supplier.name,
				supplier.phone_number,
				address.country,
				address.city,
				address.street 
			FROM supplier
			JOIN address ON supplier.address_id = address.id
			LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return []entity.SupplierWithAddress{}, fmt.Errorf("could not select suppliers: %v", err)
	}
	suppliers, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.SupplierWithAddress])
	if err != nil {
		return []entity.SupplierWithAddress{}, fmt.Errorf("could not place row to struct: %v", err)
	}
	return suppliers, nil
}

func (r *PostgresSupplierRepository) GetSupplierByID(ctx context.Context, supplierID uuid.UUID) (entity.SupplierWithAddress, error) {
	query := `
			SELECT
				supplier.id,
				supplier.name,
				supplier.phone_number,
				address.country,
				address.city,
				address.street
			FROM supplier
			JOIN address ON supplier.address_id = address.id
			WHERE supplier.id = $1`
	rows, err := r.pool.Query(ctx, query, supplierID)
	if err != nil {
		return entity.SupplierWithAddress{}, fmt.Errorf("failed to query supplier: %w", err)
	}
	supplierData, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[entity.SupplierWithAddress])
	if err != nil {
		return entity.SupplierWithAddress{}, err
	}
	return supplierData, nil
}
