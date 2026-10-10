package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sonofsun61/APIFromSpec/internal/dto"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
	"github.com/sonofsun61/APIFromSpec/internal/mapper"
)

type SupplierService interface {
	AddSupplier(ctx context.Context, newSupplierData entity.NewSupplierData, country string, city string, street string) (uuid.UUID, error)
	UpdateSupplierAddress(ctx context.Context, supplierID uuid.UUID, country string, city string, street string) error
	DeleteSupplier(ctx context.Context, supplierID uuid.UUID) error
	GetSuppliers(ctx context.Context, limit *int, offset *int) ([]entity.SupplierWithAddress, error)
	GetSupplierByID(ctx context.Context, supplierID uuid.UUID) (entity.SupplierWithAddress, error)
}

type SupplierHandler struct {
	service  SupplierService
	validate *validator.Validate
	logger *slog.Logger
}

func NewSupplierHandler(service SupplierService, validate *validator.Validate, logger *slog.Logger) *SupplierHandler {
	return &SupplierHandler{
		service:  service,
		validate: validate,
		logger: logger,
	}
}

// AddSupplier godoc
// @Summary Create a new supplier
// @Description Creates a new supplier along with a new address record, in a single transaction
// @Tags suppliers
// @Accept json
// @Produce json
// @Param request body dto.SupplierRequest true "Supplier data"
// @Success 201 {object} dto.SupplierResponse
// @Failure 400 {string} string "invalid request body or validation error"
// @Failure 500 {string} string "internal server error"
// @Router /suppliers [post]
func (h *SupplierHandler) AddSupplier(w http.ResponseWriter, r *http.Request) {
	var req dto.SupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	newSupplierData := entity.NewSupplierData{
		SupplierName: req.SupplierName,
		PhoneNumber:  req.PhoneNumber,
	}
	newSupplierID, err := h.service.AddSupplier(r.Context(), newSupplierData, req.Country, req.City, req.Street)
	if err != nil {
		writeInternalError(w, r, h.logger, err)
		return
	}
	resp := dto.SupplierResponse{
		ID: newSupplierID,
		SupplierName: req.SupplierName,
		PhoneNumber: req.PhoneNumber,
		Address: dto.AddressResponse{
			Country: req.Country,
			City: req.City,
			Street: req.Street,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// UpdateSupplierAddress godoc
// @Summary Update a supplier's address
// @Description Creates a new address record and links it to the supplier, in a single transaction
// @Tags suppliers
// @Accept json
// @Param id path string true "Supplier ID"
// @Param request body dto.UpdateAddressRequest true "New address data"
// @Success 204 "Address updated successfully"
// @Failure 400 {string} string "invalid request body, id format, or validation error"
// @Failure 404 {string} string "supplier not found"
// @Failure 500 {string} string "internal server error"
// @Router /suppliers/{id} [patch]
func (h *SupplierHandler) UpdateSupplierAddress(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var req dto.UpdateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.service.UpdateSupplierAddress(r.Context(), id, req.Country, req.City, req.Street); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "supplier not found", http.StatusNotFound)
			return
		}
		writeInternalError(w, r, h.logger, err)
		return
	}
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// DeleteSupplier godoc
// @Summary Delete a supplier by ID
// @Description Deletes a supplier by their unique identifier
// @Tags suppliers
// @Param id path string true "Supplier ID"
// @Success 204 "Supplier deleted successfully"
// @Failure 400 {string} string "invalid id format"
// @Failure 404 {string} string "supplier not found"
// @Failure 500 {string} string "internal server error"
// @Router /suppliers/{id} [delete]
func (h *SupplierHandler) DeleteSupplier(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteSupplier(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "supplier not found", http.StatusNotFound)
			return
		}
		writeInternalError(w, r, h.logger, err)
		return
	}
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// GetSuppliers godoc
// @Summary Get all suppliers
// @Description Returns a paginated list of all suppliers
// @Tags suppliers
// @Produce json
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Success 200 {array} dto.SupplierResponse
// @Failure 400 {string} string "invalid limit or offset value"
// @Failure 500 {string} string "internal server error"
// @Router /suppliers [get]
func (h *SupplierHandler) GetSuppliers(w http.ResponseWriter, r *http.Request) {
	var limit, offset *int
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		value, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		limit = &value
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		value, err := strconv.Atoi(offsetStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		offset = &value
	}
	suppliers, err := h.service.GetSuppliers(r.Context(), limit, offset)
	if err != nil {
		writeInternalError(w, r, h.logger, err)
		return
	}
	resp := mapper.SupplierWithAddressesToDTO(suppliers)
	w.Header().Set("Content-Type", "application/json")
	setCacheControl(w, "public, max-age=60")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// GetSupplierByID godoc
// @Summary Get a supplier by ID
// @Description Returns a single supplier by their unique identifier
// @Tags suppliers
// @Produce json
// @Param id path string true "Supplier ID"
// @Success 200 {object} dto.SupplierResponse
// @Failure 400 {string} string "invalid id format"
// @Failure 404 {string} string "supplier not found"
// @Failure 500 {string} string "internal server error"
// @Router /suppliers/{id} [get]
func (h *SupplierHandler) GetSupplierByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	supplierData, err := h.service.GetSupplierByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "supplier not found", http.StatusNotFound)
			return
		}
		writeInternalError(w, r, h.logger, err)
		return
	}
	resp := mapper.SupplierWithAddressToDTO(supplierData)
	w.Header().Set("Content-Type", "application/json")
	setCacheControl(w, "public, max-age=60")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
