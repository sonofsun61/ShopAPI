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
	"github.com/sonofsun61/APIFromSpec/internal/service"
)

type ProductService interface {
	CreateProduct(ctx context.Context, newProductData entity.NewProductData) (entity.Product, error)
	DecreaseStock(ctx context.Context, productID uuid.UUID, amount int) error
	GetProductByID(ctx context.Context, productID uuid.UUID) (entity.Product, error)
	GetAllProducts(ctx context.Context, limit *int, offset *int) ([]entity.Product, error)
	DeleteProductByID(ctx context.Context, productID uuid.UUID) error
}

type ProductHandler struct {
	service  ProductService
	validate *validator.Validate
	logger *slog.Logger
}

func NewProductHandler(service ProductService, validate *validator.Validate, logger *slog.Logger) *ProductHandler {
	return &ProductHandler{
		service:  service,
		validate: validate,
		logger: logger,
	}
}

// CreateProduct godoc
// @Summary Create a new product
// @Description Creates a new product; finds or creates the category by name, in a single transaction
// @Tags products
// @Accept json
// @Produce json
// @Param request body dto.CreateProductRequest true "Product data"
// @Success 201 {object} dto.ProductResponse
// @Failure 400 {string} string "invalid request body or validation error"
// @Failure 500 {string} string "internal server error"
// @Router /products [post]
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "invalid data in JSON", http.StatusBadRequest)
		return
	}
	if req.Price.IsNegative() {
		http.Error(w, "price must not be negative", http.StatusBadRequest)
		return
	}
	newProductData := entity.NewProductData{
		ProductName:    req.ProductName,
		CategoryName:   req.CategoryName,
		Price:          req.Price,
		AvailableStock: req.AvailableStock,
		SupplierID:     req.SupplierID,
	}
	product, err := h.service.CreateProduct(r.Context(), newProductData)
	if err != nil {
		writeInternalError(w, r, h.logger, err)
		return
	}
	resp := dto.ProductResponse{
		ID:             product.ID,
		ProductName:    product.ProductName,
		CategoryID:     product.CategoryID,
		Price:          product.Price,
		AvailableStock: product.AvailableStock,
		SupplierID:     product.SupplierID,
	}
	w.Header().Set("Content-Type", "application/json")
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// DecreaseStock godoc
// @Summary Decrease a product's available stock
// @Description Atomically decreases available_stock by the given amount, failing if not enough stock is available
// @Tags products
// @Accept json
// @Param id path string true "Product ID"
// @Param request body dto.DecreaseAvailableStockRequest true "Amount to decrease"
// @Success 204 "Stock decreased successfully"
// @Failure 400 {string} string "invalid request body, id format, or non-positive amount"
// @Failure 404 {string} string "product not found or insufficient stock"
// @Failure 500 {string} string "internal server error"
// @Router /products/{id}/stock [patch]
func (h *ProductHandler) DecreaseStock(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "failed to parse id", http.StatusBadRequest)
		return
	}
	var req dto.DecreaseAvailableStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err = h.validate.Struct(req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	err = h.service.DecreaseStock(r.Context(), id, req.Amount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "product not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrInvalidAmount) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeInternalError(w, r, h.logger, err)
		return
	}
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// GetProductByID godoc
// @Summary Get a product by ID
// @Description Returns a single product by its unique identifier
// @Tags products
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} dto.ProductResponse
// @Failure 400 {string} string "invalid id format"
// @Failure 404 {string} string "product not found"
// @Failure 500 {string} string "internal server error"
// @Router /products/{id} [get]
func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "failed to parse id", http.StatusBadRequest)
		return
	}
	product, err := h.service.GetProductByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "product not found", http.StatusNotFound)
			return
		}
		writeInternalError(w, r, h.logger, err)
		return
	}
	resp := mapper.ProductToDTO(product)
	w.Header().Set("Content-Type", "application/json")
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// GetAllProducts godoc
// @Summary Get all available products
// @Description Returns a paginated list of products with available_stock greater than zero
// @Tags products
// @Produce json
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Success 200 {array} dto.ProductResponse
// @Failure 400 {string} string "invalid limit or offset value"
// @Failure 500 {string} string "internal server error"
// @Router /products [get]
func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
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
	products, err := h.service.GetAllProducts(r.Context(), limit, offset)
	if err != nil {
		writeInternalError(w, r, h.logger, err)
		return
	}
	resp := mapper.ProductsToDTO(products)
	w.Header().Set("Content-Type", "application/json")
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// DeleteProductByID godoc
// @Summary Delete a product by ID
// @Description Deletes a product by its unique identifier
// @Tags products
// @Param id path string true "Product ID"
// @Success 204 "Product deleted successfully"
// @Failure 400 {string} string "invalid id format"
// @Failure 404 {string} string "product not found"
// @Failure 500 {string} string "internal server error"
// @Router /products/{id} [delete]
func (h *ProductHandler) DeleteProductByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteProductByID(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "product not found", http.StatusNotFound)
			return
		}
		writeInternalError(w, r, h.logger, err)
		return
	}
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusNoContent)
}