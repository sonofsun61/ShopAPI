package main

import (
	"fmt"
	"os"

	"github.com/sonofsun61/APIFromSpec/internal/app"
)

// @title ShopAPI
// @version 1.0
// @description REST API for a home-appliance store: clients, products, suppliers, and product images.
// @host localhost:8080
// @BasePath /api/v1
func main() {
	application := app.InitializeApp()
	if err := application.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "application stopped:", err)
		os.Exit(1)
	}
}
