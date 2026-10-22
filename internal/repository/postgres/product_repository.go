package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
)

type PostgresProductRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresProductRepository(pool *pgxpool.Pool) *PostgresProductRepository {
	return &PostgresProductRepository{
		pool: pool,
	}
}

func (r *PostgresProductRepository) CreateProduct(ctx context.Context, newProductData entity.NewProductData) (entity.Product, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return entity.Product{}, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)
	query := `
			WITH ins AS (
    			INSERT INTO category (name)
    			VALUES ($1)
    			ON CONFLICT (name) DO NOTHING
    			RETURNING id
			)
			SELECT id FROM ins
			UNION ALL
			SELECT id FROM category WHERE name = $1
			LIMIT 1;`
	var categoryID uuid.UUID
	if err := tx.QueryRow(ctx, query, newProductData.CategoryName).Scan(&categoryID); err != nil {
		return entity.Product{}, fmt.Errorf("failed to get category id: %v", err)
	}
	query = `
	INSERT INTO product (name, category_id, price, available_stock, supplier_id)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id, name, category_id, price, available_stock, last_update_date, supplier_id, image_id
			`
	rows, err := tx.Query(ctx, query,
		newProductData.ProductName,
		categoryID,
		newProductData.Price,
		newProductData.AvailableStock,
		newProductData.SupplierID)
	if err != nil {
    	return entity.Product{}, fmt.Errorf("failed to query product: %v", err)
	}
	product, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[entity.Product])
	if err != nil {
		return entity.Product{}, fmt.Errorf("failed to place row in struct: %v", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return entity.Product{}, fmt.Errorf("could not commit transaction: %v", err)
	}
	return product, nil
}

func (r *PostgresProductRepository) DecreaseStock(ctx context.Context, productID uuid.UUID, amount int) (error) {
	query := `
			UPDATE product
			SET available_stock = available_stock - $1
			WHERE id = $2 AND available_stock >= $1`
	tag, err := r.pool.Exec(ctx, query, amount, productID)
	if err != nil {
		return fmt.Errorf("failed to update product available stock: %v", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresProductRepository) GetProductByID(ctx context.Context, productID uuid.UUID) (entity.Product, error) {
	query := `
			SELECT 
				id, name, category_id,
				price, available_stock, 
				last_update_date, supplier_id, 
				image_id
			FROM product
			WHERE id = $1`
	rows, err := r.pool.Query(ctx, query, productID)
	if err != nil {
		return entity.Product{}, fmt.Errorf("failed to collect product data: %v", err)
	}
	product, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[entity.Product])
	if err != nil {
		return entity.Product{}, fmt.Errorf("failed to place row in struct: %w", err)
	}
	return product, nil
}

func (r *PostgresProductRepository) GetAllProducts(ctx context.Context, limit *int, offset *int) ([]entity.Product, error) {
	query := `
			SELECT 
				product.id,
				product.name,
				product.category_id,
				product.price,
				product.available_stock,
				product.last_update_date,
				product.supplier_id,
				product.image_id
			FROM product
			WHERE available_stock > 0
			LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return []entity.Product{}, fmt.Errorf("could not select suppliers: %v", err)
	}
	products, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.Product])
	if err != nil {
		return []entity.Product{}, fmt.Errorf("could not place row to struct: %v", err)
	}
	return products, nil
}

func (r *PostgresProductRepository) DeleteProductByID(ctx context.Context, productID uuid.UUID) (error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM product WHERE id = $1`, productID)
	if err != nil {
		return fmt.Errorf("could not delete supplier: %v", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}