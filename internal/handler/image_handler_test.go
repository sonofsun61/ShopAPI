package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
	"github.com/sonofsun61/APIFromSpec/internal/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var imageBytes = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a}

func TestAddImage(t *testing.T) {
	productID := uuid.New()
	tests := []struct {
		name        string
		id          string
		callService bool
		serviceErr  error
		wantStatus  int
	}{
		{"success", productID.String(), true, nil, http.StatusCreated},
		{"product not found", productID.String(), true, pgx.ErrNoRows, http.StatusNotFound},
		{"service error", productID.String(), true, errDBDown, http.StatusInternalServerError},
		{"invalid id", "abc", false, nil, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockImageService)
			if tt.callService {
				svc.On("AddImage", mock.Anything, imageBytes, productID).Return(tt.serviceErr)
			}
			h := handler.NewImageHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.AddImage(rec, newRequest(http.MethodPost, "/api/v1/products/"+tt.id+"/image", string(imageBytes), tt.id))

			assert.Equal(t, tt.wantStatus, rec.Code)
			svc.AssertExpectations(t)
		})
	}
}

func TestChangeImage(t *testing.T) {
	imageID := uuid.New()
	tests := []struct {
		name        string
		id          string
		callService bool
		serviceErr  error
		wantStatus  int
	}{
		{"success", imageID.String(), true, nil, http.StatusNoContent},
		{"not found", imageID.String(), true, pgx.ErrNoRows, http.StatusNotFound},
		{"service error", imageID.String(), true, errDBDown, http.StatusInternalServerError},
		{"invalid id", "abc", false, nil, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockImageService)
			if tt.callService {
				svc.On("ChangeImage", mock.Anything, imageID, imageBytes).Return(tt.serviceErr)
			}
			h := handler.NewImageHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.ChangeImage(rec, newRequest(http.MethodPatch, "/api/v1/images/"+tt.id, string(imageBytes), tt.id))

			assert.Equal(t, tt.wantStatus, rec.Code)
			svc.AssertExpectations(t)
		})
	}
}

func TestDeleteImage(t *testing.T) {
	imageID := uuid.New()
	tests := []struct {
		name        string
		id          string
		callService bool
		serviceErr  error
		wantStatus  int
	}{
		{"success", imageID.String(), true, nil, http.StatusNoContent},
		{"not found", imageID.String(), true, pgx.ErrNoRows, http.StatusNotFound},
		{"service error", imageID.String(), true, errDBDown, http.StatusInternalServerError},
		{"invalid id", "abc", false, nil, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockImageService)
			if tt.callService {
				svc.On("DeleteImage", mock.Anything, imageID).Return(tt.serviceErr)
			}
			h := handler.NewImageHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			h.DeleteImage(rec, newRequest(http.MethodDelete, "/api/v1/images/"+tt.id, "", tt.id))

			assert.Equal(t, tt.wantStatus, rec.Code)
			svc.AssertExpectations(t)
		})
	}
}

func TestGetImage(t *testing.T) {
	id := uuid.New()
	endpoints := []struct {
		name   string
		method string
		path   string
		call   func(h *handler.ImageHandler, w http.ResponseWriter, r *http.Request)
	}{
		{"by product id", "GetImageByProductID", "/api/v1/products/" + id.String() + "/image", (*handler.ImageHandler).GetImageByProductID},
		{"by image id", "GetImageByImageID", "/api/v1/images/" + id.String(), (*handler.ImageHandler).GetImageByImageID},
	}
	for _, ep := range endpoints {
		t.Run(ep.name+"/success", func(t *testing.T) {
			svc := new(mockImageService)
			svc.On(ep.method, mock.Anything, id).Return(entity.Image{ID: id, Image: imageBytes}, nil)
			h := handler.NewImageHandler(svc, validate, discardLogger)
			rec := httptest.NewRecorder()

			ep.call(h, rec, newRequest(http.MethodGet, ep.path, "", id.String()))

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "application/octet-stream", rec.Header().Get("Content-Type"))
			assert.Equal(t, "max-age=3600", rec.Header().Get("Cache-Control"))
			assert.Equal(t, imageBytes, rec.Body.Bytes())
			svc.AssertExpectations(t)
		})

		errCases := []struct {
			name       string
			id         string
			serviceErr error
			wantStatus int
		}{
			{"not found", id.String(), pgx.ErrNoRows, http.StatusNotFound},
			{"service error", id.String(), errDBDown, http.StatusInternalServerError},
			{"invalid id", "abc", nil, http.StatusBadRequest},
		}
		for _, tt := range errCases {
			t.Run(ep.name+"/"+tt.name, func(t *testing.T) {
				svc := new(mockImageService)
				if tt.wantStatus != http.StatusBadRequest {
					svc.On(ep.method, mock.Anything, id).Return(entity.Image{}, tt.serviceErr)
				}
				h := handler.NewImageHandler(svc, validate, discardLogger)
				rec := httptest.NewRecorder()

				ep.call(h, rec, newRequest(http.MethodGet, ep.path, "", tt.id))

				assert.Equal(t, tt.wantStatus, rec.Code)
				svc.AssertExpectations(t)
			})
		}
	}
}
