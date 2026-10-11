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

// Register godoc
// @Summary Register a new user
// @Description Creates a new user in the auth service and returns a signed JWT token. Does not require authentication.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Registration data"
// @Success 201 {object} dto.TokenResponse
// @Failure 400 {string} string "invalid request body or validation error"
// @Failure 500 {string} string "internal server error"
// @Failure 503 {string} string "authentication service unavailable"
// @Failure 504 {string} string "authentication service timed out"
// @Router /register [post]
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

// Login godoc
// @Summary Authenticate a user
// @Description Checks the email and password in the auth service and returns a signed JWT token. Does not require authentication.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.AuthRequest true "Login data"
// @Success 200 {object} dto.TokenResponse
// @Failure 400 {string} string "invalid request body or validation error"
// @Failure 401 {string} string "invalid credentials"
// @Failure 500 {string} string "internal server error"
// @Failure 503 {string} string "authentication service unavailable"
// @Failure 504 {string} string "authentication service timed out"
// @Router /auth [post]
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

// ResetPassword godoc
// @Summary Request a password reset
// @Description Asks the auth service to generate a new password and send it to the user. The response is the same whether or not the email exists. Does not require authentication.
// @Tags auth
// @Accept json
// @Param request body dto.ResetPasswordRequest true "Email of the user"
// @Success 204 "Reset requested"
// @Failure 400 {string} string "invalid request body or validation error"
// @Failure 500 {string} string "internal server error"
// @Failure 503 {string} string "authentication service unavailable"
// @Failure 504 {string} string "authentication service timed out"
// @Router /reset [post]
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
		h.logger.Warn("authentication rejected", "code", code.String())
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
	case codes.DeadlineExceeded:
		h.logger.Warn("auth service timed out", "code", code.String(), "err", err)
		http.Error(w, "authentication service timed out", http.StatusGatewayTimeout)
	default:
		h.logger.Error("auth service error", "code", code.String(), "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
