package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/sonofsun61/APIFromSpec/internal/dto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (string, error)
	Login(ctx context.Context, email, password string) (string, error)
	ResetPassword(ctx context.Context, email string) error
}

type AuthHandler struct {
	service  AuthService
	validate *validator.Validate
}

func NewAuthHandler(service AuthService, validate *validator.Validate) *AuthHandler {
	return &AuthHandler{
		service:  service,
		validate: validate,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	token, err := h.service.Register(r.Context(), req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dto.TokenResponse{Token: token})
}

// func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
// 	var req dto.
// 	json.NewDecoder(r.Body).Decode(req)
// }

func writeAuthError(w http.ResponseWriter, err error) {
	log.Printf("auth service error: %v", err)
	switch status.Code(err) {
	case codes.Unavailable:
		http.Error(w, "authentication service unavailable", http.StatusServiceUnavailable)
	case codes.Unauthenticated:
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
