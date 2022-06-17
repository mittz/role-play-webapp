package database

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/google/uuid"
)

type ProdDatabaseHandler struct {
	DB *sql.DB
}

func NewProdDatabaseHandler(db *sql.DB) ProdDatabaseHandler {
	return ProdDatabaseHandler{DB: db}
}

func (dbh ProdDatabaseHandler) InitDatabase() error {
	jsonFromFile, err := ioutil.ReadFile(InitDataJSONFileName)
	if err != nil {
		return err
	}

	var jsonData Blob
	if err := json.Unmarshal(jsonFromFile, &jsonData); err != nil {
		return err
	}

	db := dbh.DB

	// Don't use "IF EXISTS" as it is not supported by Spanner PGAdapter.
	queryCheckProductsTable := "SELECT * FROM products"
	queryDropProductsTables := "DROP TABLE products"
	_, err = db.Query(queryCheckProductsTable)
	tableExists := (err == nil)
	if tableExists {
		if _, err := db.Exec(queryDropProductsTables); err != nil {
			return err
		}
	}

	queryCheckUsersTable := "SELECT * FROM users"
	queryDropUsersTables := "DROP TABLE users"
	_, err = db.Query(queryCheckUsersTable)
	tableExists = (err == nil)
	if tableExists {
		if _, err := db.Exec(queryDropUsersTables); err != nil {
			return err
		}
	}

	queryCheckCheckoutsTable := "SELECT * FROM checkouts"
	queryDropCheckoutsTables := "DROP TABLE checkouts"
	_, err = db.Query(queryCheckCheckoutsTable)
	tableExists = (err == nil)
	if tableExists {
		if _, err := db.Exec(queryDropCheckoutsTables); err != nil {
			return err
		}
	}

	// Don't use "IF EXISTS" as it is not supported by Spanner PGAdapter.
	queryCreateProductsTable := `
	CREATE TABLE products (
		id bigint NOT NULL,
		name character varying(20) NOT NULL,
		price bigint NOT NULL,
		image character varying(100) NOT NULL,
		PRIMARY KEY(id)
	)
	`

	queryCreateUsersTable := `
	CREATE TABLE users (
		id bigint NOT NULL,
		name character varying(20) NOT NULL,
		PRIMARY KEY(id)
	)
	`

	queryCreateCheckoutsTable := `
	CREATE TABLE checkouts (
		id character varying(40) NOT NULL,
		user_id bigint,
		product_id bigint,
		product_quantity bigint,
		created_at date,
		PRIMARY KEY(id)
	)
	`

	if _, err := db.Exec(queryCreateProductsTable); err != nil {
		return err
	}

	if _, err := db.Exec(queryCreateUsersTable); err != nil {
		return err
	}

	if _, err := db.Exec(queryCreateCheckoutsTable); err != nil {
		return err
	}

	queryInsertProduct := "INSERT INTO products VALUES($1, $2, $3, $4)"
	for _, product := range jsonData.Products {
		if _, err := db.Exec(queryInsertProduct, product.ID, product.Name, product.Price, product.Image); err != nil {
			return err
		}
	}

	queryInsertUser := "INSERT INTO users VALUES($1, $2)"
	for _, user := range jsonData.Users {
		if _, err := db.Exec(queryInsertUser, user.ID, user.Name); err != nil {
			return err
		}
	}

	return nil
}

func (dbh ProdDatabaseHandler) GetProduct(id int) (Product, error) {
	var product Product

	db := dbh.DB
	query := "SELECT id, name, price, image FROM products WHERE id = $1"
	if err := db.QueryRow(query, id).Scan(&product.ID, &product.Name, &product.Price, &product.Image); err != nil {
		return product, err
	}

	return product, nil
}

func (dbh ProdDatabaseHandler) GetProducts() ([]Product, error) {
	var products []Product

	db := dbh.DB
	query := "SELECT id, name, price, image FROM products"
	rows, err := db.Query(query)
	if err != nil {
		return products, err
	}

	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Price, &product.Image); err != nil {
			return products, err
		}

		products = append(products, product)
	}

	return products, nil
}

func (dbh ProdDatabaseHandler) GetCheckouts(userID int) ([]Checkout, error) {
	var checkouts []Checkout

	db := dbh.DB
	query := `
	SELECT
	  checkouts.id               AS checkout_id,
	  users.id                   AS user_id,
	  users.name                 AS user_name,
	  products.id                AS product_id,
	  products.name              AS product_name,
	  products.price             AS product_price,
	  products.image             AS product_image,
	  checkouts.product_quantity AS checkout_product_quantity,
	  checkouts.created_at       AS checkout_created_at
	FROM checkouts
	LEFT JOIN users ON checkouts.user_id = users.id
	LEFT JOIN products ON checkouts.product_id = products.id
	WHERE users.id = $1
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return checkouts, err
	}

	for rows.Next() {
		var checkout Checkout
		if err := rows.Scan(&checkout.ID, &checkout.User.ID, &checkout.User.Name, &checkout.Product.ID, &checkout.Product.Name, &checkout.Product.Price, &checkout.Product.Image, &checkout.ProductQuantity, &checkout.CreatedAt); err != nil {
			return checkouts, err
		}

		checkouts = append(checkouts, checkout)
	}

	return checkouts, nil
}

func (dbh ProdDatabaseHandler) CreateCheckout(userID int, productID int, productQuantity int) (string, error) {
	uuidObj, err := uuid.NewRandom()
	checkoutID := uuidObj.String()
	if err != nil {
		return "", nil
	}

	db := dbh.DB
	query := "INSERT INTO checkouts (id, user_id, product_id, product_quantity, created_at) VALUES ($1, $2, $3, $4, $5)"
	if _, err := db.Exec(query, checkoutID, userID, productID, productQuantity, time.Now()); err != nil {
		return "", err
	}

	return checkoutID, nil
}

func (dbh ProdDatabaseHandler) GetCheckout(checkoutID string) (Checkout, error) {
	checkout := Checkout{
		Product: Product{},
		User:    User{},
	}

	// err := dbh.Conn.Joins("Product").Find(&checkout, checkoutID).Error
	db := dbh.DB
	query := `
	SELECT
	  checkouts.id,
	  users.id,
	  users.name,
	  products.id,
	  products.name,
	  products.price,
	  products.image,
	  checkouts.product_quantity,
	  checkouts.created_at
	FROM checkouts
	LEFT JOIN users ON checkouts.user_id = users.id
	LEFT JOIN products ON checkouts.product_id = products.id
	WHERE checkouts.id = $1
	`
	if err := db.QueryRow(query, checkoutID).Scan(
		&checkout.ID,
		&checkout.User.ID,
		&checkout.User.Name,
		&checkout.Product.ID,
		&checkout.Product.Name,
		&checkout.Product.Price,
		&checkout.Product.Image,
		&checkout.ProductQuantity,
		&checkout.CreatedAt,
	); err != nil {
		return checkout, err
	}

	return checkout, nil
}
