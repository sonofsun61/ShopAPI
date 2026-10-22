package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
	"github.com/sonofsun61/APIFromSpec/internal/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

var errDBDown = errors.New("db down")

func TestDecreaseStock_Success(t *testing.T) {
	repo := new(mockProductRepo)
	productID := uuid.New()
	repo.On("DecreaseStock", mock.Anything, productID, 3).Return(nil)
	svc := service.NewProductService(repo)

	err := svc.DecreaseStock(context.Background(), productID, 3)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDecreaseStock_RepoError(t *testing.T) {
	repo := new(mockProductRepo)
	productID := uuid.New()
	repo.On("DecreaseStock", mock.Anything, productID, 3).Return(errDBDown)
	svc := service.NewProductService(repo)

	err := svc.DecreaseStock(context.Background(), productID, 3)
	require.ErrorIs(t, err, errDBDown)
	repo.AssertExpectations(t)
}

func TestDecreaseStock_InvalidAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount int
	}{
		{"zero", 0},
		{"negative", -5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockProductRepo)
			svc := service.NewProductService(repo)

			err := svc.DecreaseStock(context.Background(), uuid.New(), tt.amount)
			require.ErrorIs(t, err, service.ErrInvalidAmount)
			repo.AssertNotCalled(t, "DecreaseStock", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}
