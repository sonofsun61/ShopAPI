package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sonofsun61/APIFromSpec/internal/dto"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
	"github.com/sonofsun61/APIFromSpec/internal/handler"
	"github.com/sonofsun61/APIFromSpec/internal/mapper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const validSupplierBody = `{
	"name": "Supplier",
	"phone_number": "+79999999999",
	"country": "Russia",
	"city": "Moscow",
	"street": "Arbat"
}`

func TestAddSupplier_Success(t *testing.T) {
	svc := new(mockSupplierService)
	newID := uuid.New()
	svc.On("AddSupplier", mock.Anything, entity.NewSupplierData{
		SupplierName: "Supplier",
		PhoneNumber:  "+79999999999",
	}, "Russia", "Moscow", "Arbat").Return(newID, nil)
	h := handler.NewSupplierHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.AddSupplier(rec, newRequest(http.MethodPost, "/api/v1/suppliers", validSupplierBody, ""))

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, dto.SupplierResponse{
		ID:           newID,
		SupplierName: "Supplier",
		PhoneNumber:  "+79999999999",
		Address:      dto.AddressResponse{Country: "Russia", City: "Moscow", Street: "Arbat"},
	}, decodeJSON[dto.SupplierResponse](t, rec))
	svc.AssertExpectations(t)
}

func TestAddSupplier_BadRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"invalid json", `[]`},
		{"missing name", `{"phone_number":"+79999999999","country":"Russia","city":"Moscow","street":"Arbat"}`},
		{"country too long", `{"name":"Supplier","phone_number":"+79999999999","country":"AAAAAAAAAAAAAAAAAAAAAAAAAA","city":"Moscow","street":"Arbat"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockSupplierService)
			h := handler.NewSupplierHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.AddSupplier(rec, newRequest(http.MethodPost, "/api/v1/suppliers", tt.body, ""))

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			svc.AssertNotCalled(t, "AddSupplier", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestAddSupplier_ServiceError(t *testing.T) {
	svc := new(mockSupplierService)
	svc.On("AddSupplier", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uuid.Nil, errDBDown)
	h := handler.NewSupplierHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.AddSupplier(rec, newRequest(http.MethodPost, "/api/v1/suppliers", validSupplierBody, ""))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	svc.AssertExpectations(t)
}

func TestUpdateSupplierAddress(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name        string
		id          string
		body        string
		callService bool
		serviceErr  error
		wantStatus  int
	}{
		{"success", id.String(), validAddressBody, true, nil, http.StatusNoContent},
		{"not found", id.String(), validAddressBody, true, pgx.ErrNoRows, http.StatusNotFound},
		{"service error", id.String(), validAddressBody, true, errDBDown, http.StatusInternalServerError},
		{"invalid id", "abc", validAddressBody, false, nil, http.StatusBadRequest},
		{"invalid json", id.String(), `{"country":`, false, nil, http.StatusBadRequest},
		{"empty country", id.String(), `{"country":"","city":"Moscow","street":"Arbat"}`, false, nil, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockSupplierService)
			if tt.callService {
				svc.On("UpdateSupplierAddress", mock.Anything, id, "Russia", "Moscow", "Arbat").Return(tt.serviceErr)
			}
			h := handler.NewSupplierHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.UpdateSupplierAddress(rec, newRequest(http.MethodPatch, "/api/v1/suppliers/"+tt.id, tt.body, tt.id))

			assert.Equal(t, tt.wantStatus, rec.Code)
			svc.AssertExpectations(t)
		})
	}
}

func TestDeleteSupplier(t *testing.T) {
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
			svc := new(mockSupplierService)
			if tt.callService {
				svc.On("DeleteSupplier", mock.Anything, id).Return(tt.serviceErr)
			}
			h := handler.NewSupplierHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.DeleteSupplier(rec, newRequest(http.MethodDelete, "/api/v1/suppliers/"+tt.id, "", tt.id))

			assert.Equal(t, tt.wantStatus, rec.Code)
			svc.AssertExpectations(t)
		})
	}
}

func TestGetSuppliers_Success(t *testing.T) {
	svc := new(mockSupplierService)
	suppliers := []entity.SupplierWithAddress{{ID: uuid.New(), SupplierName: "Supplier", City: "Moscow"}}
	svc.On("GetSuppliers", mock.Anything, intPtr(10), intPtr(0)).Return(suppliers, nil)
	h := handler.NewSupplierHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.GetSuppliers(rec, newRequest(http.MethodGet, "/api/v1/suppliers?limit=10&offset=0", "", ""))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "public, max-age=60", rec.Header().Get("Cache-Control"))
	assert.Equal(t, mapper.SupplierWithAddressesToDTO(suppliers), decodeJSON[[]dto.SupplierResponse](t, rec))
	svc.AssertExpectations(t)
}

func TestGetSuppliers_Errors(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		callService bool
		wantStatus  int
	}{
		{"invalid limit", "?limit=ten", false, http.StatusBadRequest},
		{"invalid offset", "?offset=-", false, http.StatusBadRequest},
		{"service error", "", true, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockSupplierService)
			if tt.callService {
				svc.On("GetSuppliers", mock.Anything, mock.Anything, mock.Anything).Return([]entity.SupplierWithAddress(nil), errDBDown)
			}
			h := handler.NewSupplierHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.GetSuppliers(rec, newRequest(http.MethodGet, "/api/v1/suppliers"+tt.query, "", ""))

			assert.Equal(t, tt.wantStatus, rec.Code)
			svc.AssertExpectations(t)
		})
	}
}

func TestGetSupplierByID_Success(t *testing.T) {
	svc := new(mockSupplierService)
	supplier := entity.SupplierWithAddress{ID: uuid.New(), SupplierName: "Supplier", PhoneNumber: "+79999999999"}
	svc.On("GetSupplierByID", mock.Anything, supplier.ID).Return(supplier, nil)
	h := handler.NewSupplierHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.GetSupplierByID(rec, newRequest(http.MethodGet, "/api/v1/suppliers/"+supplier.ID.String(), "", supplier.ID.String()))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "public, max-age=60", rec.Header().Get("Cache-Control"))
	assert.Equal(t, mapper.SupplierWithAddressToDTO(supplier), decodeJSON[dto.SupplierResponse](t, rec))
	svc.AssertExpectations(t)
}

func TestGetSupplierByID_Errors(t *testing.T) {
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
			svc := new(mockSupplierService)
			if tt.callService {
				svc.On("GetSupplierByID", mock.Anything, id).Return(entity.SupplierWithAddress{}, tt.serviceErr)
			}
			h := handler.NewSupplierHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.GetSupplierByID(rec, newRequest(http.MethodGet, "/api/v1/suppliers/"+tt.id, "", tt.id))

			assert.Equal(t, tt.wantStatus, rec.Code)
			svc.AssertExpectations(t)
		})
	}
}
