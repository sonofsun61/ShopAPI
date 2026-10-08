package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	_ "github.com/sonofsun61/APIFromSpec/docs"
	"github.com/sonofsun61/APIFromSpec/internal/middleware"
	"github.com/sonofsun61/APIFromSpec/internal/routes"
	httpSwagger "github.com/swaggo/http-swagger"
)

func SetUpRouter(clientHandler *ClientHandler, supplierHandler *SupplierHandler,
	productHandler *ProductHandler, imageHandler *ImageHandler,
	authHandler *AuthHandler, tokenValidator middleware.TokenValidator) http.Handler {
	r := chi.NewRouter()
	r.Get("/swagger/*", httpSwagger.WrapHandler)
	r.Post(routes.RegisterBasePath, authHandler.Register)
	r.Post(routes.AuthBasePath, authHandler.Login)
	r.Post(routes.ResetPasswordBasePath, authHandler.ResetPassword)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(tokenValidator))
		r.Route(routes.ClientsBasePath, func(r chi.Router) {
			r.Post("/", clientHandler.CreateClient)
			r.Delete("/{id}", clientHandler.DeleteClientByID)
			r.Get("/", clientHandler.GetClients)
			r.Patch("/{id}", clientHandler.UpdateClientAddress)
		})
		r.Route(routes.SuppliersBasePath, func(r chi.Router) {
			r.Post("/", supplierHandler.AddSupplier)
			r.Patch("/{id}", supplierHandler.UpdateSupplierAddress)
			r.Delete("/{id}", supplierHandler.DeleteSupplier)
			r.Get("/", supplierHandler.GetSuppliers)
			r.Get("/{id}", supplierHandler.GetSupplierByID)
		})
		r.Route(routes.ProductsBasePath, func(r chi.Router) {
			r.Post("/", productHandler.CreateProduct)
			r.Patch("/{id}/stock", productHandler.DecreaseStock)
			r.Get("/{id}", productHandler.GetProductByID)
			r.Get("/", productHandler.GetAllProducts)
			r.Delete("/{id}", productHandler.DeleteProductByID)
			r.Post("/{id}/image", imageHandler.AddImage)
			r.Get("/{id}/image", imageHandler.GetImageByProductID)
		})
		r.Route(routes.ImagesBasePath, func(r chi.Router) {
			r.Patch("/{id}", imageHandler.ChangeImage)
			r.Delete("/{id}", imageHandler.DeleteImage)
			r.Get("/{id}", imageHandler.GetImageByImageID)
		})
	})
	return r
}
