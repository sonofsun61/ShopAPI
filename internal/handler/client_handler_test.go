package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sonofsun61/APIFromSpec/internal/dto"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
	"github.com/sonofsun61/APIFromSpec/internal/handler"
	"github.com/sonofsun61/APIFromSpec/internal/mapper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const validClientBody = `{
	"client_name": "Ivan",
	"client_surname": "Ivanov",
	"birthday": "1990-05-01T00:00:00Z",
	"gender": "male",
	"country": "Russia",
	"city": "Moscow",
	"street": "Tverskaya"
}`

const validAddressBody = `{"country":"Russia","city":"Moscow","street":"Arbat"}`

func TestCreateClient_Success(t *testing.T) {
	svc := new(mockClientService)
	newID := uuid.New()
	birthday := time.Date(1990, time.May, 1, 0, 0, 0, 0, time.UTC)
	svc.On("CreateClient", mock.Anything, entity.NewClientData{
		ClientName:    "Ivan",
		ClientSurname: "Ivanov",
		Birthday:      birthday,
		Gender:        "male",
	}, "Russia", "Moscow", "Tverskaya").Return(newID, nil)
	h := handler.NewClientHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.CreateClient(rec, newRequest(http.MethodPost, "/api/v1/clients", validClientBody, ""))

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	resp := decodeJSON[dto.ClientResponse](t, rec)
	assert.Equal(t, newID, resp.ID)
	assert.Equal(t, "Ivan", resp.ClientName)
	assert.Equal(t, "Ivanov", resp.ClientSurname)
	assert.True(t, birthday.Equal(resp.Birthday))
	assert.Equal(t, "male", resp.Gender)
	assert.Equal(t, dto.AddressResponse{Country: "Russia", City: "Moscow", Street: "Tverskaya"}, resp.Address)
	svc.AssertExpectations(t)
}

func TestCreateClient_BadRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"invalid json", `{"client_name":`},
		{"unknown gender", `{"client_name":"Ivan","client_surname":"Ivanov","birthday":"1990-05-01T00:00:00Z","gender":"other","country":"Russia","city":"Moscow","street":"Tverskaya"}`},
		{"birthday in future", `{"client_name":"Ivan","client_surname":"Ivanov","birthday":"2999-01-01T00:00:00Z","gender":"male","country":"Russia","city":"Moscow","street":"Tverskaya"}`},
		{"missing city", `{"client_name":"Ivan","client_surname":"Ivanov","birthday":"1990-05-01T00:00:00Z","gender":"male","country":"Russia","street":"Tverskaya"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockClientService)
			h := handler.NewClientHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.CreateClient(rec, newRequest(http.MethodPost, "/api/v1/clients", tt.body, ""))

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			svc.AssertNotCalled(t, "CreateClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestCreateClient_ServiceError(t *testing.T) {
	svc := new(mockClientService)
	svc.On("CreateClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uuid.Nil, errDBDown)
	h := handler.NewClientHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.CreateClient(rec, newRequest(http.MethodPost, "/api/v1/clients", validClientBody, ""))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), errDBDown.Error(), "internal errors must not leak to the client")
	svc.AssertExpectations(t)
}

func TestDeleteClientByID(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name       string
		id         string
		serviceErr error
		wantStatus int
	}{
		{"success", id.String(), nil, http.StatusNoContent},
		{"not found", id.String(), pgx.ErrNoRows, http.StatusNotFound},
		{"service error", id.String(), errDBDown, http.StatusInternalServerError},
		{"invalid id", "not-a-uuid", nil, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockClientService)
			if tt.wantStatus != http.StatusBadRequest {
				svc.On("DeleteClientByID", mock.Anything, id).Return(tt.serviceErr)
			}
			h := handler.NewClientHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.DeleteClientByID(rec, newRequest(http.MethodDelete, "/api/v1/clients/"+tt.id, "", tt.id))

			assert.Equal(t, tt.wantStatus, rec.Code)
			svc.AssertExpectations(t)
		})
	}
}

func TestGetClients_ByNameAndSurname(t *testing.T) {
	svc := new(mockClientService)
	clients := []entity.ClientWithAddress{{ID: uuid.New(), ClientName: "Ivan", ClientSurname: "Ivanov", Country: "Russia"}}
	svc.On("GetClientByNameAndSurname", mock.Anything, "Ivan", "Ivanov").Return(clients, nil)
	h := handler.NewClientHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.GetClients(rec, newRequest(http.MethodGet, "/api/v1/clients?name=Ivan&surname=Ivanov", "", ""))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, mapper.ClientsWithAddressesToDTO(clients), decodeJSON[[]dto.ClientResponse](t, rec))
	svc.AssertExpectations(t)
	svc.AssertNotCalled(t, "GetAllClients", mock.Anything, mock.Anything, mock.Anything)
}

func TestGetClients_All(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantLimit  *int
		wantOffset *int
	}{
		{"no params", "", nil, nil},
		{"limit and offset", "?limit=10&offset=20", intPtr(10), intPtr(20)},
		{"only limit", "?limit=5", intPtr(5), nil},
		{"only name", "?name=Ivan", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockClientService)
			svc.On("GetAllClients", mock.Anything, tt.wantLimit, tt.wantOffset).Return([]entity.ClientWithAddress{}, nil)
			h := handler.NewClientHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.GetClients(rec, newRequest(http.MethodGet, "/api/v1/clients"+tt.query, "", ""))

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.JSONEq(t, `[]`, rec.Body.String())
			svc.AssertExpectations(t)
		})
	}
}

func TestGetClients_InvalidPagination(t *testing.T) {
	for _, query := range []string{"?limit=abc", "?offset=1.5"} {
		t.Run(query, func(t *testing.T) {
			svc := new(mockClientService)
			h := handler.NewClientHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.GetClients(rec, newRequest(http.MethodGet, "/api/v1/clients"+query, "", ""))

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			svc.AssertNotCalled(t, "GetAllClients", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestGetClients_ServiceError(t *testing.T) {
	svc := new(mockClientService)
	svc.On("GetAllClients", mock.Anything, mock.Anything, mock.Anything).Return([]entity.ClientWithAddress(nil), errDBDown)
	h := handler.NewClientHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.GetClients(rec, newRequest(http.MethodGet, "/api/v1/clients", "", ""))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	svc.AssertExpectations(t)
}

func TestUpdateClientAddress(t *testing.T) {
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
		{"invalid id", "123", validAddressBody, false, nil, http.StatusBadRequest},
		{"invalid json", id.String(), `{`, false, nil, http.StatusBadRequest},
		{"missing street", id.String(), `{"country":"Russia","city":"Moscow"}`, false, nil, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockClientService)
			if tt.callService {
				svc.On("UpdateClientAddress", mock.Anything, id, "Russia", "Moscow", "Arbat").Return(tt.serviceErr)
			}
			h := handler.NewClientHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.UpdateClientAddress(rec, newRequest(http.MethodPatch, "/api/v1/clients/"+tt.id, tt.body, tt.id))

			assert.Equal(t, tt.wantStatus, rec.Code)
			svc.AssertExpectations(t)
		})
	}
}
