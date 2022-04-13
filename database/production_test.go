package database

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewMockDatabaseHandler() (DatabaseHandler, sqlmock.Sqlmock, error) {
	dbh := NewProdDatabaseHandler()

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}

	gdbconn, err := gorm.Open(postgres.New(postgres.Config{
		Conn: mockDB,
	}), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	dbh.DB.Conn = gdbconn

	return dbh, mock, nil
}

func TestReadProperties(t *testing.T) {
	dirname, err := ioutil.TempDir("", "scstore-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirname)

	jsonFile, err := ioutil.TempFile(dirname, "*.json")
	if err != nil {
		t.Fatal(err)
	}

	b, err := json.Marshal(&Database{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "postgres"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := jsonFile.WriteString(string(b)); err != nil {
		t.Fatal(err)
	}

	dbh := NewProdDatabaseHandler()
	err = dbh.ReadProperties(jsonFile.Name())

	assert.Equal(t, err, nil)
	assert.Equal(t, dbh.DB.Host, "localhost")
}

func TestGetProduct(t *testing.T) {
	mdb, mock, err := NewMockDatabaseHandler()
	if err != nil {
		t.Fatal(err)
	}

	p := Product{Model: gorm.Model{ID: uint(1)}, Name: "hunters-race-Vk3QiwyrAUA", Price: uint(150), Image: "/assets/hunters-race-Vk3QiwyrAUA-unsplash.jpg"}

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "products" WHERE "products"."id" = $1 AND "products"."deleted_at" IS NULL ORDER BY "products"."id" LIMIT 1`)).
		WithArgs(p.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "price", "image"}).
			AddRow(p.ID, p.Name, p.Price, p.Image))

	product, err := mdb.GetProduct(p.ID)

	assert.Equal(t, err, nil)
	assert.Equal(t, product, p)
}

func TestGetProducts(t *testing.T) {
	mdb, mock, err := NewMockDatabaseHandler()
	if err != nil {
		t.Fatal(err)
	}

	productID1, productID2 := uint(1), uint(2)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "products" WHERE "products"."deleted_at" IS NULL`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(productID1).
			AddRow(productID2))

	products, err := mdb.GetProducts()
	assert.Equal(t, err, nil)
	assert.Equal(t, len(products), 2)
}

func TestGetCheckouts(t *testing.T) {
	mdb, mock, err := NewMockDatabaseHandler()
	if err != nil {
		t.Fatal(err)
	}

	userID := uint(1)
	checkout1 := Checkout{UserID: userID, ProductID: uint(1)}
	checkout2 := Checkout{UserID: userID, ProductID: uint(2)}

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "checkouts" WHERE user_id = $1 AND "checkouts"."deleted_at" IS NULL`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "product_id"}).
			AddRow(userID, checkout1.ProductID).
			AddRow(userID, checkout2.ProductID))

	checkouts, err := mdb.GetCheckouts(userID)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(checkouts), 2)
	assert.Equal(t, checkouts[0], checkout1)
	assert.Equal(t, checkouts[1], checkout2)
}
