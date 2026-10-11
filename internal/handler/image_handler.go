package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
)

type ImageService interface {
	AddImage(ctx context.Context, imageBytes []byte, productID uuid.UUID) error
	ChangeImage(ctx context.Context, imageID uuid.UUID, imageBytes []byte) error
	DeleteImage(ctx context.Context, imageID uuid.UUID) error
	GetImageByProductID(ctx context.Context, productID uuid.UUID) (entity.Image, error)
	GetImageByImageID(ctx context.Context, imageID uuid.UUID) (entity.Image, error)
}

type ImageHandler struct {
	service  ImageService
	validate *validator.Validate
	logger *slog.Logger
}

func NewImageHandler(service ImageService, validate *validator.Validate, logger *slog.Logger) *ImageHandler {
	return &ImageHandler{
		service:  service,
		validate: validate,
		logger: logger,
	}
}

// AddImage godoc
// @Summary Add an image to a product
// @Description Uploads raw image bytes and links them to the given product, in a single transaction
// @Tags images
// @Accept octet-stream
// @Param id path string true "Product ID"
// @Success 201 "Image added successfully"
// @Failure 400 {string} string "invalid id format or failed to read request body"
// @Failure 404 {string} string "product not found"
// @Failure 500 {string} string "internal server error"
// @Router /products/{id}/image [post]
func (h *ImageHandler) AddImage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	imageBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err = h.service.AddImage(r.Context(), imageBytes, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		writeInternalError(w, r, h.logger, err)
		return
	}
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusCreated)
}

// ChangeImage godoc
// @Summary Replace an existing image
// @Description Replaces the bytes of an existing image by its ID
// @Tags images
// @Accept octet-stream
// @Param id path string true "Image ID"
// @Success 204 "Image updated successfully"
// @Failure 400 {string} string "invalid id format or failed to read request body"
// @Failure 404 {string} string "image not found"
// @Failure 500 {string} string "internal server error"
// @Router /images/{id} [patch]
func (h *ImageHandler) ChangeImage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	imageBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err = h.service.ChangeImage(r.Context(), id, imageBytes); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		writeInternalError(w, r, h.logger, err)
		return
	}
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// DeleteImage godoc
// @Summary Delete an image by ID
// @Description Deletes an image by its unique identifier; the owning product's image_id is set to null
// @Tags images
// @Param id path string true "Image ID"
// @Success 204 "Image deleted successfully"
// @Failure 400 {string} string "invalid id format"
// @Failure 404 {string} string "image not found"
// @Failure 500 {string} string "internal server error"
// @Router /images/{id} [delete]
func (h *ImageHandler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err = h.service.DeleteImage(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		writeInternalError(w, r, h.logger, err)
		return
	}
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// GetImageByProductID godoc
// @Summary Get a product's image
// @Description Returns the raw image bytes linked to the given product
// @Tags images
// @Produce octet-stream
// @Param id path string true "Product ID"
// @Success 200 {file} file
// @Failure 400 {string} string "invalid id format"
// @Failure 404 {string} string "product not found or has no image"
// @Failure 500 {string} string "internal server error"
// @Router /products/{id}/image [get]
func (h *ImageHandler) GetImageByProductID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "failed to parse id", http.StatusBadRequest)
		return
	}
	image, err := h.service.GetImageByProductID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		writeInternalError(w, r, h.logger, err)
		return
	}
	resp := image.Image
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=image")
	setCacheControl(w, "max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// GetImageByImageID godoc
// @Summary Get an image by ID
// @Description Returns the raw image bytes by the image's unique identifier
// @Tags images
// @Produce octet-stream
// @Param id path string true "Image ID"
// @Success 200 {file} file
// @Failure 400 {string} string "invalid id format"
// @Failure 404 {string} string "image not found"
// @Failure 500 {string} string "internal server error"
// @Router /images/{id} [get]
func (h *ImageHandler) GetImageByImageID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "failed to parse id", http.StatusBadRequest)
		return
	}
	image, err := h.service.GetImageByImageID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		writeInternalError(w, r, h.logger, err)
		return
	}
	resp := image.Image
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=image")
	setCacheControl(w, "max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}
