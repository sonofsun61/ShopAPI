package mapper

import (
	"github.com/sonofsun61/APIFromSpec/internal/dto"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
	"github.com/sonofsun61/APIFromSpec/internal/routes"
)

func SupplierWithAddressToDTO(supplier entity.SupplierWithAddress) dto.SupplierResponse {
	address := dto.AddressResponse{
		Country: supplier.Country,
		City:    supplier.City,
		Street:  supplier.Street,
	}
	links := map[string]dto.Link{
		"self": {
			Href:   routes.SuppliersBasePath + "/" + supplier.ID.String(),
			Method: "GET",
		},
		"update_address": {
			Href:   routes.SuppliersBasePath + "/" + supplier.ID.String(),
			Method: "PATCH",
		},
		"delete": {
			Href:   routes.SuppliersBasePath + "/" + supplier.ID.String(),
			Method: "DELETE",
		},
	}
	supplierData := dto.SupplierResponse{
		ID:           supplier.ID,
		SupplierName: supplier.SupplierName,
		PhoneNumber:  supplier.PhoneNumber,
		Address:      address,
		Links:        links,
	}
	return supplierData
}

func SupplierWithAddressesToDTO(suppliers []entity.SupplierWithAddress) []dto.SupplierResponse {
	resp := make([]dto.SupplierResponse, len(suppliers))
	for i := range suppliers {
		supplier := SupplierWithAddressToDTO(suppliers[i])
		resp[i] = supplier
	}
	return resp
}
