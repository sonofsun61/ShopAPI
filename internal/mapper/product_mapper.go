package mapper

import (
	"github.com/sonofsun61/APIFromSpec/internal/dto"
	"github.com/sonofsun61/APIFromSpec/internal/entity"
	"github.com/sonofsun61/APIFromSpec/internal/routes"
)

func ProductToDTO(product entity.Product) dto.ProductResponse {
	links := map[string]dto.Link{
		"self": {
			Href:   routes.ProductsBasePath + "/" + product.ID.String(),
			Method: "GET",
		},
		"decrease_stock": {
			Href:   routes.ProductsBasePath + "/" + product.ID.String(),
			Method: "PATCH",
		},
		"delete": {
			Href:   routes.ProductsBasePath + "/" + product.ID.String(),
			Method: "DELETE",
		},
		"add_image": {
			Href:   routes.ProductsBasePath + "/" + product.ID.String() + "/image",
			Method: "POST",
		},
	}
	productData := dto.ProductResponse{
		ID:             product.ID,
		ProductName:    product.ProductName,
		CategoryID:     product.CategoryID,
		Price:          product.Price,
		AvailableStock: product.AvailableStock,
		SupplierID:     product.SupplierID,
		Links:          links,
	}
	return productData
}

func ProductsToDTO(products []entity.Product) []dto.ProductResponse {
	resp := make([]dto.ProductResponse, len(products))
	for i := range products {
		product := ProductToDTO(products[i])
		resp[i] = product
	}
	return resp
}
