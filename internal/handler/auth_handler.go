package handler

import (
	"context"
	"encoding/json"
	"log/slog"
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
	logger   *slog.Logger
}

func NewAuthHandler(service AuthService, validate *validator.Validate, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		service:  service,
		validate: validate,
		logger:   logger,
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
		h.writeAuthError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dto.TokenResponse{Token: token})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	token, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dto.TokenResponse{Token: token})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.service.ResetPassword(r.Context(), req.Email); err != nil {
		h.writeAuthError(w, err)
		return
	}
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) writeAuthError(w http.ResponseWriter, err error) {
	code := status.Code(err)
	switch status.Code(err) {
	case codes.Unavailable:
		h.logger.Error("auth service unavailable", "code", code.String(), "err", err)
		http.Error(w, "authentication service unavailable", http.StatusServiceUnavailable)
	case codes.Unauthenticated:
		h.logger.Error("authentication rejected", "code", code.String())
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
	default:
		h.logger.Error("auth service error", "code", code.String(), "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
