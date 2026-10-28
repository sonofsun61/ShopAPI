package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/sonofsun61/APIFromSpec/internal/dto"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
	"github.com/sonofsun61/APIFromSpec/internal/handler"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	discardLogger = slog.New(slog.DiscardHandler)
	validate      = validator.New()
	errDBDown     = errors.New("db down")
)

func newRequest(method, target, body, id string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if id != "" {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}
	return req
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var v T
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&v))
	return v
}

func intPtr(v int) *int {
	return &v
}

type mockAuthService struct {
	mock.Mock
}

var _ handler.AuthService = (*mockAuthService)(nil)

func (m *mockAuthService) Register(ctx context.Context, req dto.RegisterRequest) (string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Error(1)
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.Error(1)
}

func (m *mockAuthService) ResetPassword(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

type mockClientService struct {
	mock.Mock
}

var _ handler.ClientService = (*mockClientService)(nil)

func (m *mockClientService) CreateClient(ctx context.Context, newClientData entity.NewClientData, country string, city string, street string) (uuid.UUID, error) {
	args := m.Called(ctx, newClientData, country, city, street)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockClientService) DeleteClientByID(ctx context.Context, clientID uuid.UUID) error {
	args := m.Called(ctx, clientID)
	return args.Error(0)
}

func (m *mockClientService) GetClientByNameAndSurname(ctx context.Context, name string, surname string) ([]entity.ClientWithAddress, error) {
	args := m.Called(ctx, name, surname)
	return args.Get(0).([]entity.ClientWithAddress), args.Error(1)
}

func (m *mockClientService) GetAllClients(ctx context.Context, limit *int, offset *int) ([]entity.ClientWithAddress, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]entity.ClientWithAddress), args.Error(1)
}

func (m *mockClientService) UpdateClientAddress(ctx context.Context, clientID uuid.UUID, country string, city string, street string) error {
	args := m.Called(ctx, clientID, country, city, street)
	return args.Error(0)
}

type mockSupplierService struct {
	mock.Mock
}

var _ handler.SupplierService = (*mockSupplierService)(nil)

func (m *mockSupplierService) AddSupplier(ctx context.Context, newSupplierData entity.NewSupplierData, country string, city string, street string) (uuid.UUID, error) {
	args := m.Called(ctx, newSupplierData, country, city, street)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockSupplierService) UpdateSupplierAddress(ctx context.Context, supplierID uuid.UUID, country string, city string, street string) error {
	args := m.Called(ctx, supplierID, country, city, street)
	return args.Error(0)
}

func (m *mockSupplierService) DeleteSupplier(ctx context.Context, supplierID uuid.UUID) error {
	args := m.Called(ctx, supplierID)
	return args.Error(0)
}

func (m *mockSupplierService) GetSuppliers(ctx context.Context, limit *int, offset *int) ([]entity.SupplierWithAddress, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]entity.SupplierWithAddress), args.Error(1)
}

func (m *mockSupplierService) GetSupplierByID(ctx context.Context, supplierID uuid.UUID) (entity.SupplierWithAddress, error) {
	args := m.Called(ctx, supplierID)
	return args.Get(0).(entity.SupplierWithAddress), args.Error(1)
}

type mockProductService struct {
	mock.Mock
}

var _ handler.ProductService = (*mockProductService)(nil)

func (m *mockProductService) CreateProduct(ctx context.Context, newProductData entity.NewProductData) (entity.Product, error) {
	args := m.Called(ctx, newProductData)
	return args.Get(0).(entity.Product), args.Error(1)
}

func (m *mockProductService) DecreaseStock(ctx context.Context, productID uuid.UUID, amount int) error {
	args := m.Called(ctx, productID, amount)
	return args.Error(0)
}

func (m *mockProductService) GetProductByID(ctx context.Context, productID uuid.UUID) (entity.Product, error) {
	args := m.Called(ctx, productID)
	return args.Get(0).(entity.Product), args.Error(1)
}

func (m *mockProductService) GetAllProducts(ctx context.Context, limit *int, offset *int) ([]entity.Product, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]entity.Product), args.Error(1)
}

func (m *mockProductService) DeleteProductByID(ctx context.Context, productID uuid.UUID) error {
	args := m.Called(ctx, productID)
	return args.Error(0)
}

type mockImageService struct {
	mock.Mock
}

var _ handler.ImageService = (*mockImageService)(nil)

func (m *mockImageService) AddImage(ctx context.Context, imageBytes []byte, productID uuid.UUID) error {
	args := m.Called(ctx, imageBytes, productID)
	return args.Error(0)
}

func (m *mockImageService) ChangeImage(ctx context.Context, imageID uuid.UUID, imageBytes []byte) error {
	args := m.Called(ctx, imageID, imageBytes)
	return args.Error(0)
}

func (m *mockImageService) DeleteImage(ctx context.Context, imageID uuid.UUID) error {
	args := m.Called(ctx, imageID)
	return args.Error(0)
}

func (m *mockImageService) GetImageByProductID(ctx context.Context, productID uuid.UUID) (entity.Image, error) {
	args := m.Called(ctx, productID)
	return args.Get(0).(entity.Image), args.Error(1)
}

func (m *mockImageService) GetImageByImageID(ctx context.Context, imageID uuid.UUID) (entity.Image, error) {
	args := m.Called(ctx, imageID)
	return args.Get(0).(entity.Image), args.Error(1)
}
