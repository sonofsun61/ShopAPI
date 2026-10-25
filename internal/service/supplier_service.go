package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
)

type SupplierRepository interface {
	AddSupplier(ctx context.Context, newSupplierData entity.NewSupplierData, country string, city string, street string) (uuid.UUID, error)
	UpdateSupplierAddress(ctx context.Context, supplierID uuid.UUID, country string, city string, street string) error
	DeleteSupplier(ctx context.Context, supplierID uuid.UUID) error
	GetSuppliers(ctx context.Context, limit *int, offset *int) ([]entity.SupplierWithAddress, error)
	GetSupplierByID(ctx context.Context, supplierID uuid.UUID) (entity.SupplierWithAddress, error)
}

type SupplierService struct {
	repo SupplierRepository
}

func NewSupplierService(repo SupplierRepository) *SupplierService {
	return &SupplierService{repo: repo}
}

func (s *SupplierService) AddSupplier(ctx context.Context, newSupplierData entity.NewSupplierData, country string, city string, street string) (uuid.UUID, error) {
	return s.repo.AddSupplier(ctx, newSupplierData, country, city, street)
}

func (s *SupplierService) UpdateSupplierAddress(ctx context.Context, supplierID uuid.UUID, country string, city string, street string) error {
	return s.repo.UpdateSupplierAddress(ctx, supplierID, country, city, street)
}

func (s *SupplierService) DeleteSupplier(ctx context.Context, supplierID uuid.UUID) error {
	return s.repo.DeleteSupplier(ctx, supplierID)
}

func (s *SupplierService) GetSuppliers(ctx context.Context, limit *int, offset *int) ([]entity.SupplierWithAddress, error) {
	return s.repo.GetSuppliers(ctx, limit, offset)
}

func (s *SupplierService) GetSupplierByID(ctx context.Context, supplierID uuid.UUID) (entity.SupplierWithAddress, error) {
	return s.repo.GetSupplierByID(ctx, supplierID)
}
