package database

import (
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

	dbh.Conn = gdbconn

	return dbh, mock, nil
}

func TestNewProdDatabaseHandler(t *testing.T) {
	dbh := NewProdDatabaseHandler()
	assert.NotNil(t, dbh)
}

func TestOpenDatabase(t *testing.T) {
}

func TestInitDatabase(t *testing.T) {

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

	assert.Nil(t, err)
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
	assert.Nil(t, err)
	assert.Equal(t, 2, len(products))
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
		`SELECT "checkouts"."id","checkouts"."created_at","checkouts"."updated_at","checkouts"."deleted_at","checkouts"."user_id","checkouts"."product_id","checkouts"."product_quantity","User"."id" AS "User__id","User"."created_at" AS "User__created_at","User"."updated_at" AS "User__updated_at","User"."deleted_at" AS "User__deleted_at","User"."name" AS "User__name","User"."password" AS "User__password","Product"."id" AS "Product__id","Product"."created_at" AS "Product__created_at","Product"."updated_at" AS "Product__updated_at","Product"."deleted_at" AS "Product__deleted_at","Product"."name" AS "Product__name","Product"."price" AS "Product__price","Product"."image" AS "Product__image" FROM "checkouts" LEFT JOIN "users" "User" ON "checkouts"."user_id" = "User"."id" AND "User"."deleted_at" IS NULL LEFT JOIN "products" "Product" ON "checkouts"."product_id" = "Product"."id" AND "Product"."deleted_at" IS NULL WHERE "checkouts"."deleted_at" IS NULL`)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "product_id"}).
			AddRow(userID, checkout1.ProductID).
			AddRow(userID, checkout2.ProductID))

	checkouts, err := mdb.GetCheckouts(userID)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(checkouts))
	assert.Equal(t, checkout1, checkouts[0])
	assert.Equal(t, checkout2, checkouts[1])
}

func TestCreateCheckout(t *testing.T) {

}

func TestGetCheckout(t *testing.T) {
	mdb, mock, err := NewMockDatabaseHandler()
	if err != nil {
		t.Fatal(err)
	}

	checkoutID, userID, productID, productQuantity := uint(1), uint(1), uint(1), uint(1)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT "checkouts"."id","checkouts"."created_at","checkouts"."updated_at","checkouts"."deleted_at","checkouts"."user_id","checkouts"."product_id","checkouts"."product_quantity","Product"."id" AS "Product__id","Product"."created_at" AS "Product__created_at","Product"."updated_at" AS "Product__updated_at","Product"."deleted_at" AS "Product__deleted_at","Product"."name" AS "Product__name","Product"."price" AS "Product__price","Product"."image" AS "Product__image" FROM "checkouts" LEFT JOIN "products" "Product" ON "checkouts"."product_id" = "Product"."id" AND "Product"."deleted_at" IS NULL WHERE "checkouts"."id" = $1 AND "checkouts"."deleted_at" IS NULL`)).
		WithArgs(checkoutID).
		WillReturnRows(sqlmock.NewRows([]string{"checkout_id", "user_id", "product_id", "product_quantity"}).AddRow(checkoutID, userID, productID, productQuantity))

	_, err = mdb.GetCheckout(checkoutID)
	assert.Nil(t, err)
}
