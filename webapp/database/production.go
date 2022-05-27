package database

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"gorm.io/gorm"
)

type ProdDatabaseHandler struct {
	Conn *gorm.DB
}

func NewProdDatabaseHandler(conn *gorm.DB) ProdDatabaseHandler {
	return ProdDatabaseHandler{Conn: conn}
}

func (dbh ProdDatabaseHandler) InitDatabase() error {
	jsonFromFile, err := ioutil.ReadFile(InitDataJSONFileName)
	if err != nil {
		log.Fatal(err)
	}

	var jsonData Blob
	err = json.Unmarshal(jsonFromFile, &jsonData)
	if err != nil {
		log.Fatal(err)
	}

	// Drop table if exists
	if err := dbh.Conn.Migrator().DropTable(&Product{}, &User{}, &Checkout{}); err != nil {
		return err
	}
	if err := dbh.Conn.Migrator().CreateTable(&Product{}, &User{}, &Checkout{}); err != nil {
		return err
	}
	if result := dbh.Conn.Model(&Product{}).Create(jsonData.Products); result.Error != nil {
		return result.Error
	}
	if result := dbh.Conn.Model(&User{}).Create(jsonData.Users); result.Error != nil {
		return result.Error
	}

	return nil
}

func (dbh ProdDatabaseHandler) GetProduct(id uint) (Product, error) {
	var product Product

	result := dbh.Conn.First(&product, id)
	if result.Error != nil {
		return product, result.Error
	}

	return product, nil
}

func (dbh ProdDatabaseHandler) GetProducts() ([]Product, error) {
	var products []Product

	result := dbh.Conn.Find(&products)
	if result.Error != nil {
		return products, result.Error
	}

	return products, nil
}

func (dbh ProdDatabaseHandler) GetCheckouts(userID uint) ([]Checkout, error) {
	var checkouts []Checkout

	// TODO: Consider if index handling is required
	err := dbh.Conn.Joins("User").Joins("Product").Find(&checkouts).Where("users.id =?", userID).Error

	return checkouts, err
}

func (dbh ProdDatabaseHandler) CreateCheckout(userID uint, productID uint, productQuantity uint) (uint, error) {
	checkout := Checkout{
		UserID:          userID,
		ProductID:       productID,
		ProductQuantity: productQuantity,
	}
	result := dbh.Conn.Create(&checkout)

	return checkout.ID, result.Error
}

func (dbh ProdDatabaseHandler) GetCheckout(checkoutID uint) (Checkout, error) {
	var checkout Checkout

	err := dbh.Conn.Joins("Product").Find(&checkout, checkoutID).Error

	return checkout, err
}
