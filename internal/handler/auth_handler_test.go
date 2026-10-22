package handler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sonofsun61/APIFromSpec/internal/dto"
	"github.com/sonofsun61/APIFromSpec/internal/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const validRegisterBody = `{
	"email": "user@mail.com",
	"name": "Ivan",
	"surname": "Ivanov",
	"phone_number": "+79999999999",
	"password": "password123"
}`

func grpcErr(code codes.Code) error {
	return fmt.Errorf("failed to call auth service: %w", status.Error(code, "boom"))
}

func TestRegister_Success(t *testing.T) {
	svc := new(mockAuthService)
	svc.On("Register", mock.Anything, dto.RegisterRequest{
		Email:       "user@mail.com",
		Name:        "Ivan",
		Surname:     "Ivanov",
		PhoneNumber: "+79999999999",
		Password:    "password123",
	}).Return("token-123", nil)
	h := handler.NewAuthHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.Register(rec, newRequest(http.MethodPost, "/api/v1/register", validRegisterBody, ""))

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	assert.Equal(t, dto.TokenResponse{Token: "token-123"}, decodeJSON[dto.TokenResponse](t, rec))
	svc.AssertExpectations(t)
}

func TestRegister_BadRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"invalid json", `{"email":`},
		{"invalid email", `{"email":"not-an-email","name":"Ivan","surname":"Ivanov","phone_number":"+79999999999","password":"password123"}`},
		{"short password", `{"email":"user@mail.com","name":"Ivan","surname":"Ivanov","phone_number":"+79999999999","password":"short"}`},
		{"phone with letters", `{"email":"user@mail.com","name":"Ivan","surname":"Ivanov","phone_number":"+7999abc9999","password":"password123"}`},
		{"missing name", `{"email":"user@mail.com","surname":"Ivanov","phone_number":"+79999999999","password":"password123"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockAuthService)
			h := handler.NewAuthHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.Register(rec, newRequest(http.MethodPost, "/api/v1/register", tt.body, ""))

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			svc.AssertNotCalled(t, "Register", mock.Anything, mock.Anything)
		})
	}
}

func TestLogin_AuthServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"unauthenticated", grpcErr(codes.Unauthenticated), http.StatusUnauthorized},
		{"unavailable", grpcErr(codes.Unavailable), http.StatusServiceUnavailable},
		{"deadline exceeded", grpcErr(codes.DeadlineExceeded), http.StatusGatewayTimeout},
		{"internal", grpcErr(codes.Internal), http.StatusInternalServerError},
		{"not a grpc error", errDBDown, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockAuthService)
			svc.On("Login", mock.Anything, "user@mail.com", "password123").Return("", tt.err)
			h := handler.NewAuthHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.Login(rec, newRequest(http.MethodPost, "/api/v1/auth", `{"email":"user@mail.com","password":"password123"}`, ""))

			assert.Equal(t, tt.wantStatus, rec.Code)
			svc.AssertExpectations(t)
		})
	}
}

func TestRegister_AuthServiceError(t *testing.T) {
	svc := new(mockAuthService)
	svc.On("Register", mock.Anything, mock.Anything).Return("", grpcErr(codes.Unavailable))
	h := handler.NewAuthHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.Register(rec, newRequest(http.MethodPost, "/api/v1/register", validRegisterBody, ""))

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	svc.AssertExpectations(t)
}

func TestLogin_Success(t *testing.T) {
	svc := new(mockAuthService)
	svc.On("Login", mock.Anything, "user@mail.com", "password123").Return("token-123", nil)
	h := handler.NewAuthHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.Login(rec, newRequest(http.MethodPost, "/api/v1/auth", `{"email":"user@mail.com","password":"password123"}`, ""))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	assert.Equal(t, dto.TokenResponse{Token: "token-123"}, decodeJSON[dto.TokenResponse](t, rec))
	svc.AssertExpectations(t)
}

func TestLogin_BadRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"invalid json", `not json`},
		{"missing password", `{"email":"user@mail.com"}`},
		{"invalid email", `{"email":"user","password":"password123"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockAuthService)
			h := handler.NewAuthHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.Login(rec, newRequest(http.MethodPost, "/api/v1/auth", tt.body, ""))

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			svc.AssertNotCalled(t, "Login", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestResetPassword_Success(t *testing.T) {
	svc := new(mockAuthService)
	svc.On("ResetPassword", mock.Anything, "user@mail.com").Return(nil)
	h := handler.NewAuthHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.ResetPassword(rec, newRequest(http.MethodPost, "/api/v1/reset", `{"email":"user@mail.com"}`, ""))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
	svc.AssertExpectations(t)
}

func TestResetPassword_BadRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"invalid json", `{`},
		{"invalid email", `{"email":"user"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockAuthService)
			h := handler.NewAuthHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.ResetPassword(rec, newRequest(http.MethodPost, "/api/v1/reset", tt.body, ""))

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			svc.AssertNotCalled(t, "ResetPassword", mock.Anything, mock.Anything)
		})
	}
}

func TestResetPassword_AuthServiceError(t *testing.T) {
	svc := new(mockAuthService)
	svc.On("ResetPassword", mock.Anything, "user@mail.com").Return(grpcErr(codes.DeadlineExceeded))
	h := handler.NewAuthHandler(svc, validate, discardLogger)
	rec := httptest.NewRecorder()

	h.ResetPassword(rec, newRequest(http.MethodPost, "/api/v1/reset", `{"email":"user@mail.com"}`, ""))

	assert.Equal(t, http.StatusGatewayTimeout, rec.Code)
	svc.AssertExpectations(t)
}
