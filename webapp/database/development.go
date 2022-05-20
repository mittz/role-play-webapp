package database

import "gorm.io/gorm"

type DevDatabaseHandler struct {
	DB *Database
}

func NewDevDatabaseHandler() DevDatabaseHandler {
	return DevDatabaseHandler{DB: &Database{}}
}

func (dbh DevDatabaseHandler) OpenDatabase() error {
	return nil
}

func (dbh DevDatabaseHandler) InitDatabase() error {
	return nil
}

func (dbh DevDatabaseHandler) GetProduct(id uint) (Product, error) {
	product := Product{
		Model: gorm.Model{ID: uint(1)}, Name: "product1", Price: uint(100), Image: "image/product1.png",
	}

	return product, nil
}

func (dbh DevDatabaseHandler) GetProducts() ([]Product, error) {
	products := []Product{
		{Model: gorm.Model{ID: uint(1)}, Name: "product1", Price: uint(100), Image: "image/product1.png"},
		{Model: gorm.Model{ID: uint(2)}, Name: "product2", Price: uint(200), Image: "image/product2.png"},
	}

	return products, nil
}

func (dbh DevDatabaseHandler) GetCheckouts(userID uint) ([]Checkout, error) {
	checkouts := []Checkout{
		{
			UserID:          userID,
			Product:         Product{Price: uint(100), Image: "image/product1.png"},
			ProductQuantity: uint(111),
		},
		{
			UserID:          userID,
			Product:         Product{Price: uint(200), Image: "image/product2.png"},
			ProductQuantity: uint(222),
		},
	}

	return checkouts, nil
}

func (dbh DevDatabaseHandler) CreateCheckout(userID uint, productID uint, productQuantity uint) (uint, error) {
	return 0, nil
}

func (dbh DevDatabaseHandler) GetCheckout(checkoutID uint) (Checkout, error) {
	checkout := Checkout{
		Model: gorm.Model{ID: checkoutID}, Product: Product{Name: "product1", Image: "image/product1.png"}, ProductQuantity: uint(111),
	}

	return checkout, nil
}
