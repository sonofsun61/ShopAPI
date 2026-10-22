package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/sonofsun61/APIFromSpec/internal/dto"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
	"github.com/sonofsun61/APIFromSpec/internal/handler"
	"github.com/sonofsun61/APIFromSpec/internal/mapper"
	"github.com/sonofsun61/APIFromSpec/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var testSupplierID = uuid.MustParse("6f1c2b8e-0b1a-4c7e-9a52-3d4e5f6a7b8c")

func productBody(price string, stock string) string {
	return `{"product_name":"Fridge","category_name":"Kitchen","price":"` + price +
		`","available_stock":` + stock + `,"supplier_id":"` + testSupplierID.String() + `"}`
}

func TestCreateProduct_Success(t *testing.T) {
	svc := new(mockProductService)
	product := entity.Product{
		ID:             uuid.New(),
		ProductName:    "Fridge",
		CategoryID:     uuid.New(),
		Price:          decimal.RequireFromString("499.99"),
		AvailableStock: 5,
		SupplierID:     testSupplierID,
	}
	svc.On("CreateProduct", mock.Anything, mock.MatchedBy(func(d entity.NewProductData) bool {
		return d.ProductName == "Fridge" &&
			d.CategoryName == "Kitchen" &&
			d.Price.Equal(decimal.RequireFromString("499.99")) &&
			d.AvailableStock == 5 &&
			d.SupplierID == testSupplierID
	})).Return(product, nil)
	h := handler.NewProductHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.CreateProduct(rec, newRequest(http.MethodPost, "/api/v1/products", productBody("499.99", "5"), ""))

	assert.Equal(t, http.StatusCreated, rec.Code)
	resp := decodeJSON[dto.ProductResponse](t, rec)
	assert.Equal(t, product.ID, resp.ID)
	assert.Equal(t, product.CategoryID, resp.CategoryID)
	assert.True(t, product.Price.Equal(resp.Price))
	assert.Equal(t, product.AvailableStock, resp.AvailableStock)
	svc.AssertExpectations(t)
}

func TestCreateProduct_BadRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"invalid json", `{"product_name"`},
		{"negative price", productBody("-1", "5")},
		{"negative stock", productBody("10", "-1")},
		{"missing name", `{"category_name":"Kitchen","price":"10","available_stock":5}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockProductService)
			h := handler.NewProductHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.CreateProduct(rec, newRequest(http.MethodPost, "/api/v1/products", tt.body, ""))

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			svc.AssertNotCalled(t, "CreateProduct", mock.Anything, mock.Anything)
		})
	}
}

func TestCreateProduct_ServiceError(t *testing.T) {
	svc := new(mockProductService)
	svc.On("CreateProduct", mock.Anything, mock.Anything).Return(entity.Product{}, errDBDown)
	h := handler.NewProductHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.CreateProduct(rec, newRequest(http.MethodPost, "/api/v1/products", productBody("10", "1"), ""))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	svc.AssertExpectations(t)
}

func TestDecreaseStock(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name        string
		id          string
		body        string
		callService bool
		serviceErr  error
		wantStatus  int
	}{
		{"success", id.String(), `{"amount":3}`, true, nil, http.StatusNoContent},
		{"not found or insufficient stock", id.String(), `{"amount":3}`, true, pgx.ErrNoRows, http.StatusNotFound},
		{"invalid amount from service", id.String(), `{"amount":3}`, true, service.ErrInvalidAmount, http.StatusBadRequest},
		{"service error", id.String(), `{"amount":3}`, true, errDBDown, http.StatusInternalServerError},
		{"invalid id", "abc", `{"amount":3}`, false, nil, http.StatusBadRequest},
		{"invalid json", id.String(), `{"amount":"three"}`, false, nil, http.StatusBadRequest},
		{"zero amount", id.String(), `{"amount":0}`, false, nil, http.StatusBadRequest},
		{"negative amount", id.String(), `{"amount":-2}`, false, nil, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockProductService)
			if tt.callService {
				svc.On("DecreaseStock", mock.Anything, id, 3).Return(tt.serviceErr)
			}
			h := handler.NewProductHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.DecreaseStock(rec, newRequest(http.MethodPatch, "/api/v1/products/"+tt.id+"/stock", tt.body, tt.id))

			assert.Equal(t, tt.wantStatus, rec.Code)
			svc.AssertExpectations(t)
		})
	}
}

func TestGetProductByID_Success(t *testing.T) {
	svc := new(mockProductService)
	product := entity.Product{ID: uuid.New(), ProductName: "Fridge", Price: decimal.NewFromInt(100), AvailableStock: 2}
	svc.On("GetProductByID", mock.Anything, product.ID).Return(product, nil)
	h := handler.NewProductHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.GetProductByID(rec, newRequest(http.MethodGet, "/api/v1/products/"+product.ID.String(), "", product.ID.String()))

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeJSON[dto.ProductResponse](t, rec)
	want := mapper.ProductToDTO(product)
	assert.Equal(t, want.ID, resp.ID)
	assert.Equal(t, want.Links, resp.Links)
	svc.AssertExpectations(t)
}

func TestGetProductByID_Errors(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name        string
		id          string
		callService bool
		serviceErr  error
		wantStatus  int
	}{
		{"not found", id.String(), true, pgx.ErrNoRows, http.StatusNotFound},
		{"service error", id.String(), true, errDBDown, http.StatusInternalServerError},
		{"invalid id", "abc", false, nil, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockProductService)
			if tt.callService {
				svc.On("GetProductByID", mock.Anything, id).Return(entity.Product{}, tt.serviceErr)
			}
			h := handler.NewProductHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.GetProductByID(rec, newRequest(http.MethodGet, "/api/v1/products/"+tt.id, "", tt.id))

			assert.Equal(t, tt.wantStatus, rec.Code)
			svc.AssertExpectations(t)
		})
	}
}

func TestGetAllProducts(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		callService bool
		limit       *int
		offset      *int
		serviceErr  error
		wantStatus  int
	}{
		{"no params", "", true, nil, nil, nil, http.StatusOK},
		{"with pagination", "?limit=2&offset=4", true, intPtr(2), intPtr(4), nil, http.StatusOK},
		{"service error", "", true, nil, nil, errDBDown, http.StatusInternalServerError},
		{"invalid limit", "?limit=x", false, nil, nil, nil, http.StatusBadRequest},
		{"invalid offset", "?offset=x", false, nil, nil, nil, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockProductService)
			if tt.callService {
				svc.On("GetAllProducts", mock.Anything, tt.limit, tt.offset).Return([]entity.Product{}, tt.serviceErr)
			}
			h := handler.NewProductHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.GetAllProducts(rec, newRequest(http.MethodGet, "/api/v1/products"+tt.query, "", ""))

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				assert.JSONEq(t, `[]`, rec.Body.String())
			}
			svc.AssertExpectations(t)
		})
	}
}

func TestDeleteProductByID(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name        string
		id          string
		callService bool
		serviceErr  error
		wantStatus  int
	}{
		{"success", id.String(), true, nil, http.StatusNoContent},
		{"not found", id.String(), true, pgx.ErrNoRows, http.StatusNotFound},
		{"service error", id.String(), true, errDBDown, http.StatusInternalServerError},
		{"invalid id", "abc", false, nil, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockProductService)
			if tt.callService {
				svc.On("DeleteProductByID", mock.Anything, id).Return(tt.serviceErr)
			}
			h := handler.NewProductHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.DeleteProductByID(rec, newRequest(http.MethodDelete, "/api/v1/products/"+tt.id, "", tt.id))

			assert.Equal(t, tt.wantStatus, rec.Code)
			svc.AssertExpectations(t)
		})
	}
}
