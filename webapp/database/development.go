package database

import (
	"database/sql"
)

type DevDatabaseHandler struct {
	DB *sql.DB
}

func NewDevDatabaseHandler(db *sql.DB) DevDatabaseHandler {
	return DevDatabaseHandler{DB: db}
}

func (dbh DevDatabaseHandler) InitDatabase() error {
	return nil
}

func (dbh DevDatabaseHandler) GetProduct(id int) (Product, error) {
	product := Product{
		ID: 1, Name: "product1", Price: 100, Image: "image/product1.png",
	}

	return product, nil
}

func (dbh DevDatabaseHandler) GetProducts() ([]Product, error) {
	products := []Product{
		{ID: 1, Name: "product1", Price: 100, Image: "image/product1.png"},
		{ID: 2, Name: "product2", Price: 200, Image: "image/product2.png"},
	}

	return products, nil
}

func (dbh DevDatabaseHandler) GetCheckouts(userID int) ([]Checkout, error) {
	checkouts := []Checkout{
		{
			User:            User{ID: userID},
			Product:         Product{Price: 100, Image: "image/product1.png"},
			ProductQuantity: 111,
		},
		{
			User:            User{ID: userID},
			Product:         Product{Price: 200, Image: "image/product2.png"},
			ProductQuantity: 222,
		},
	}

	return checkouts, nil
}

func (dbh DevDatabaseHandler) CreateCheckout(userID int, productID int, productQuantity int) (string, error) {
	return "", nil
}

func (dbh DevDatabaseHandler) GetCheckout(checkoutID string) (Checkout, error) {
	checkout := Checkout{
		ID: checkoutID, Product: Product{Name: "product1", Image: "image/product1.png"}, ProductQuantity: 111,
	}

	return checkout, nil
}
