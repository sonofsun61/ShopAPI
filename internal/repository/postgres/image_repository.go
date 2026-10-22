package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
)

type PostgresImageRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresImageRepository(pool *pgxpool.Pool) *PostgresImageRepository {
	return &PostgresImageRepository{
		pool: pool,
	}
}

func (r *PostgresImageRepository) AddImage(ctx context.Context, imageBytes []byte, productID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	query := `Insert into images (image) values ($1) returning id`
	var newImageID uuid.UUID
	if err = tx.QueryRow(ctx, query, imageBytes).Scan(&newImageID); err != nil {
		return fmt.Errorf("failed to scan image id into variable: %v", err)
	}

	query = `Update product set image_id = $1 where id = $2`
	tag, err := tx.Exec(ctx, query, newImageID, productID)
	if err != nil {
		return fmt.Errorf("failed to connect product and image: %v", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}
	return nil
}

func (r *PostgresImageRepository) ChangeImage(ctx context.Context, imageID uuid.UUID, imageBytes []byte) error {
	query := `Update images set image = $1 where id = $2`
	tag, err := r.pool.Exec(ctx, query, imageBytes, imageID)
	if err != nil {
		return fmt.Errorf("failed to update image: %v", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresImageRepository) DeleteImage(ctx context.Context, imageID uuid.UUID) error {
	query := `Delete from images where id = $1`
	tag, err := r.pool.Exec(ctx, query, imageID)
	if err != nil {
		return fmt.Errorf("failed to delete image: %v", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresImageRepository) GetImageByProductID(ctx context.Context, productID uuid.UUID) (entity.Image, error) {
	query := `
			Select images.id, images.image
			from images
			join product on images.id = product.image_id
			where product.id = $1`
	rows, err := r.pool.Query(ctx, query, productID)
	if err != nil {
		return entity.Image{}, fmt.Errorf("failed to get image data: %v", err)
	}
	imageData, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[entity.Image])
	if err != nil {
		return entity.Image{}, fmt.Errorf("failed to collect image data into structure: %w", err)
	}
	return imageData, nil
}

func (r *PostgresImageRepository) GetImageByImageID(ctx context.Context, imageID uuid.UUID) (entity.Image, error) {
	query := `Select image from images where id = $1`
	var imageBytes []byte
	if err := r.pool.QueryRow(ctx, query, imageID).Scan(&imageBytes); err != nil {
		return entity.Image{}, fmt.Errorf("failed to get image: %w", err)
	}
	return entity.Image{
		ID:    imageID,
		Image: imageBytes,
	}, nil
}
