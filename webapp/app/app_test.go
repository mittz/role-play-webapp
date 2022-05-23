package app

import (
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/mittz/role-play-webapp/webapp/database"
	"github.com/stretchr/testify/assert" // TODO: Replace this with gopkg.in/check.v1
)

var dbDevHandler database.DatabaseHandler

const testAssetsDir = "./assets"
const testTemplatesDirMatch = "./templates/*"

func TestGetProductEndpoint(t *testing.T) {
	router := SetupRouter(dbDevHandler, testAssetsDir, testTemplatesDirMatch)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/product/1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "product1")
	assert.Contains(t, w.Body.String(), "$100")
}

func TestGetProductsEndpoint(t *testing.T) {
	router := SetupRouter(dbDevHandler, testAssetsDir, testTemplatesDirMatch)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/products", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "product1")
	assert.Contains(t, w.Body.String(), "$100")
	assert.Contains(t, w.Body.String(), "/product/1")
	assert.Contains(t, w.Body.String(), "image/product1")
	assert.Contains(t, w.Body.String(), "product2")
	assert.Contains(t, w.Body.String(), "$200")
	assert.Contains(t, w.Body.String(), "/product/2")
	assert.Contains(t, w.Body.String(), "image/product2")
}

func TestGetCheckoutsEndpoint(t *testing.T) {
	router := SetupRouter(dbDevHandler, testAssetsDir, testTemplatesDirMatch)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/checkouts", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "product1")
	assert.Contains(t, w.Body.String(), "111")
	assert.Contains(t, w.Body.String(), "product2")
	assert.Contains(t, w.Body.String(), "222")
}

func TestPostCheckoutEndpoint(t *testing.T) {
	router := SetupRouter(dbDevHandler, testAssetsDir, testTemplatesDirMatch)

	values := url.Values{}
	productID := "1"
	productQuantity := "111"
	values.Add("product_id", productID)
	values.Add("product_quantity", productQuantity)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/checkout", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	assert.Equal(t, 202, w.Code)
	assert.Contains(t, w.Body.String(), "product1")
	assert.Contains(t, w.Body.String(), "image/product1.png")
	assert.Contains(t, w.Body.String(), productQuantity)
}

func TestMain(m *testing.M) {
	dbHandler, err := database.NewDatabaseHandler("development")
	if err != nil {
		log.Fatal(err)
	}
	dbDevHandler = dbHandler

	os.Exit(m.Run())
}
