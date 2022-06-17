package database

import (
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func NewMockDatabaseHandler() (DatabaseHandler, sqlmock.Sqlmock, error) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}

	dbh := NewProdDatabaseHandler(mockDB)

	return dbh, mock, nil
}

func TestNewProdDatabaseHandler(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	assert.Nil(t, err)

	dbh := NewProdDatabaseHandler(mockDB)
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

	p := Product{ID: 1, Name: "", Price: 150, Image: "/assets/hunters-race-Vk3QiwyrAUA-unsplash.jpg"}

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, name, price, image FROM products WHERE id = $1`)).
		WithArgs(p.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "price", "image"}).
			AddRow(p.ID, p.Name, p.Price, p.Image))

	product, err := mdb.GetProduct(p.ID)

	assert.Nil(t, err)
	assert.Equal(t, p, product)
}

func TestGetProducts(t *testing.T) {
	mdb, mock, err := NewMockDatabaseHandler()
	if err != nil {
		t.Fatal(err)
	}

	p1 := Product{ID: 1, Name: "Product00001", Price: 150, Image: "product00001.jpg"}
	p2 := Product{ID: 2, Name: "Product00002", Price: 200, Image: "product00002.jpg"}

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, name, price, image FROM products`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "price", "image"}).
			AddRow(p1.ID, p1.Name, p1.Price, p1.Image).
			AddRow(p2.ID, p2.Name, p2.Price, p2.Image))

	products, err := mdb.GetProducts()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(products))
	assert.Equal(t, p1, products[0])
	assert.Equal(t, p2, products[1])
}

func TestGetCheckouts(t *testing.T) {
	mdb, mock, err := NewMockDatabaseHandler()
	if err != nil {
		t.Fatal(err)
	}

	userID, userName := 1, "user00001"
	checkout1 := Checkout{
		ID: "dummy-checkout-00001",
		User: User{
			ID:   userID,
			Name: userName,
		},
		Product: Product{
			ID:    1,
			Name:  "product00001",
			Price: 100,
			Image: "product00001.png",
		},
		ProductQuantity: 1,
		CreatedAt:       time.Now(),
	}

	checkout2 := Checkout{
		ID: "dummy-checkout-00002",
		User: User{
			ID:   userID,
			Name: userName,
		},
		Product: Product{
			ID:    1,
			Name:  "product00002",
			Price: 200,
			Image: "product00002.png",
		},
		ProductQuantity: 2,
		CreatedAt:       time.Now(),
	}

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT checkouts.id AS checkout_id, users.id AS user_id, users.name AS user_name, products.id AS product_id, products.name AS product_name, products.price AS product_price, products.image AS product_image, checkouts.product_quantity AS checkout_product_quantity, checkouts.created_at AS checkout_created_at FROM checkouts LEFT JOIN users ON checkouts.user_id = users.id LEFT JOIN products ON checkouts.product_id = products.id WHERE users.id = $1`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"checkout_id", "user_id", "user_name", "product_id", "product_name", "product_price", "product_image", "checkout_product_quantity", "checkout_created_at"}).
			AddRow(checkout1.ID, checkout1.User.ID, checkout1.User.Name, checkout1.Product.ID, checkout1.Product.Name, checkout1.Product.Price, checkout1.Product.Image, checkout1.ProductQuantity, checkout1.CreatedAt).
			AddRow(checkout2.ID, checkout2.User.ID, checkout2.User.Name, checkout2.Product.ID, checkout2.Product.Name, checkout2.Product.Price, checkout2.Product.Image, checkout2.ProductQuantity, checkout2.CreatedAt))

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

	checkout := Checkout{
		ID: "dummy-checkout-00001",
		User: User{
			ID:   1,
			Name: "user00001",
		},
		Product: Product{
			ID:    1,
			Name:  "Product00001",
			Price: 100,
			Image: "product00001.png",
		},
		ProductQuantity: 1,
		CreatedAt:       time.Now(),
	}

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT checkouts.id, users.id, users.name, products.id, products.name, products.price, products.image, checkouts.product_quantity, checkouts.created_at FROM checkouts LEFT JOIN users ON checkouts.user_id = users.id LEFT JOIN products ON checkouts.product_id = products.id WHERE checkouts.id = $1`)).
		WithArgs(checkout.ID).
		WillReturnRows(sqlmock.NewRows([]string{"checkout_id", "user_id", "user_name", "product_id", "product_name", "product_price", "product_image", "product_quantity", "created_at"}).
			AddRow(checkout.ID, checkout.User.ID, checkout.User.Name, checkout.Product.ID, checkout.Product.Name, checkout.Product.Price, checkout.Product.Image, checkout.ProductQuantity, checkout.CreatedAt))

	c, err := mdb.GetCheckout(checkout.ID)
	assert.Nil(t, err)
	assert.Equal(t, checkout, c)
}
