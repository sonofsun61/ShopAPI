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

type ClientService interface {
	CreateClient(ctx context.Context, newClientData entity.NewClientData, country string, city string, street string) (uuid.UUID, error)
	DeleteClientByID(ctx context.Context, clientID uuid.UUID) error
	GetClientByNameAndSurname(ctx context.Context, name string, surname string) ([]entity.ClientWithAddress, error)
	GetAllClients(ctx context.Context, limit *int, offset *int) ([]entity.ClientWithAddress, error)
	UpdateClientAddress(ctx context.Context, clientID uuid.UUID, country string, city string, street string) error
}

type ClientHandler struct {
	service  ClientService
	validate *validator.Validate
	logger *slog.Logger
}

func NewClientHandler(service ClientService, validate *validator.Validate, logger *slog.Logger) *ClientHandler {
	return &ClientHandler{
		service:  service,
		validate: validate,
		logger: logger,
	}
}

// Create client godoc
// @Summary Create a new client
// @Description Creates a new client along with a new address record, in a single transaction.
// @Tags clients
// @Accept json
// @Produce json
// @Param request body dto.CreateClientRequest true "Client data"
// @Success 201 {object} dto.ClientResponse
// @Failure 500 {string} string "internal server error"
// @Router /clients [post]
func (h *ClientHandler) CreateClient(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	newClientData := entity.NewClientData{
		ClientName: req.ClientName,
		ClientSurname: req.ClientSurname,
		Birthday: req.Birthday,
		Gender: req.Gender,
	}
	newID, err := h.service.CreateClient(r.Context(), newClientData, req.Country, req.City, req.Street)
	if err != nil {
		writeInternalError(w, r, h.logger, err)
		return
	}
	resp := dto.ClientResponse{
		ID: newID,
		ClientName: req.ClientName,
		ClientSurname: req.ClientSurname,
		Birthday: req.Birthday,
		Gender: req.Gender,
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

// Create client godoc
// @Summary Delete an existing client
// @Description Deletes an existing client without deleting his address
// @Tags clients
// @Param id path string true "Client ID"
// @Success 204 "Client deleted successfully"
// @Failure 400 {string} string "Invalid id format"
// @Failure 404 {string} string "Client not found"
// @Failure 500 {string} string "Internal server error"
// @Router /clients/{id} [delete]
func (h *ClientHandler) DeleteClientByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteClientByID(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "client not found", http.StatusNotFound)
			return
		}
		writeInternalError(w, r, h.logger, err)
		return
	}
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// GetClients godoc
// @Summary Get clients
// @Description Returns clients filtered by name and surname, or a paginated list of all clients if no filter is given
// @Tags clients
// @Produce json
// @Param name query string false "Client first name"
// @Param surname query string false "Client surname"
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Success 200 {array} dto.ClientResponse
// @Failure 400 {string} string "invalid limit or offset value"
// @Failure 500 {string} string "internal server error"
// @Router /clients [get]
func (h *ClientHandler) GetClients(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	surname := r.URL.Query().Get("surname")
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
	var clients []entity.ClientWithAddress
	var err error
	if name != "" && surname != "" {
		clients, err = h.service.GetClientByNameAndSurname(r.Context(), name, surname)
	} else {
		clients, err = h.service.GetAllClients(r.Context(), limit, offset)
	}
	if err != nil {
		writeInternalError(w, r, h.logger, err)
		return
	}
	resp := mapper.ClientsWithAddressesToDTO(clients)
	w.Header().Set("Content-Type", "application/json")
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// UpdateClientAddress godoc
// @Summary Update a client's address
// @Description Creates a new address record and links it to the client, in a single transaction
// @Tags clients
// @Accept json
// @Param id path string true "Client ID"
// @Param request body dto.UpdateAddressRequest true "New address data"
// @Success 204 "Address updated successfully"
// @Failure 400 {string} string "invalid request body, id format, or validation error"
// @Failure 404 {string} string "client not found"
// @Failure 500 {string} string "internal server error"
// @Router /clients/{id} [patch]
func (h *ClientHandler) UpdateClientAddress(w http.ResponseWriter, r *http.Request) {
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
	if err := h.service.UpdateClientAddress(r.Context(), id, req.Country, req.City, req.Street); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "client not found", http.StatusNotFound)
			return
		}
		writeInternalError(w, r, h.logger, err)
		return
	}
	setCacheControl(w, "no-store")
	w.WriteHeader(http.StatusNoContent)
}