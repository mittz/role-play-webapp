package database

import (
	"fmt"

	"gorm.io/gorm"
)

type DatabaseHandler interface {
	ReadProperties(filename string) error
	OpenDatabase() error
	InitDatabase() error
	GetProduct(id uint) (Product, error)
	GetProducts() ([]Product, error)
	GetCheckouts(userID uint) ([]Checkout, error)
	CreateCheckout(userID uint, productID uint, productQuantity uint) (uint, error)
	GetCheckout(checkoutID uint) (Checkout, error)
}

const InitDataJSONFileName = "initdata.json"

type Database struct {
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	Conn     *gorm.DB
}

type Product struct {
	gorm.Model
	Name  string
	Price uint
	Image string
}

type User struct {
	gorm.Model
	Name     string
	Password string
}

type Checkout struct {
	gorm.Model
	UserID          uint
	User            User
	ProductID       uint
	Product         Product
	ProductQuantity uint
}

type Blob struct {
	Products []Product `json:"products"`
	Users    []User    `json:"users"`
}

func NewDatabaseHandler(environment string) (DatabaseHandler, error) {
	switch environment {
	case "production":
		return NewProdDatabaseHandler(), nil
	case "development":
		return NewDevDatabaseHandler(), nil
	default:
		return nil, fmt.Errorf("environment: %s is not supported", environment)
	}
}
