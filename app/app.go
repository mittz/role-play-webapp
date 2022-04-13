package app

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mittz/scaledce-role-play-series/webapp/database"
)

var dbHandler database.DatabaseHandler

func postInitEndpoint(c *gin.Context) {
	if err := dbHandler.InitDatabase(); err != nil {
		c.String(http.StatusInternalServerError, "%v", err)
	}

	c.String(http.StatusAccepted, "Initiatilized data.")
}

func getCheckoutsEndpoint(c *gin.Context) {
	userID := uint(1)
	checkouts, err := dbHandler.GetCheckouts(userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "%v", err)
	}

	c.HTML(http.StatusOK, "checkouts.tmpl", gin.H{
		"title":     "Checkouts",
		"checkouts": checkouts,
	})
}

func postCheckoutEndpoint(c *gin.Context) {
	userID := 1

	productID, err := strconv.Atoi(c.PostForm("product_id"))
	if err != nil {
		c.String(http.StatusInternalServerError, "%v", err)
	}

	productQuantity, err := strconv.Atoi(c.PostForm("product_quantity"))
	if err != nil {
		c.String(http.StatusInternalServerError, "%v", err)
	}

	checkoutID, err := dbHandler.CreateCheckout(uint(userID), uint(productID), uint(productQuantity))
	if err != nil {
		c.String(http.StatusInternalServerError, "%v", err)
	}

	checkout, err := dbHandler.GetCheckout(checkoutID)
	if err != nil {
		c.String(http.StatusInternalServerError, "%v", err)
	}

	c.HTML(http.StatusAccepted, "checkout.tmpl", gin.H{
		"title":    "Checkout",
		"checkout": checkout,
	})
}

func getProductEndpoint(c *gin.Context) {
	productID, err := strconv.Atoi(c.Param("product_id"))
	if err != nil {
		c.String(http.StatusInternalServerError, "%v", err)
	}

	product, err := dbHandler.GetProduct(uint(productID))
	if err != nil {
		c.String(http.StatusInternalServerError, "%v", err)
	}

	c.HTML(http.StatusOK, "product.tmpl", gin.H{
		"title":   "Product",
		"product": product,
	})
}

func getProductsEndpoint(c *gin.Context) {
	products, err := dbHandler.GetProducts()
	if err != nil {
		c.String(http.StatusInternalServerError, "%v", err)
	}

	c.HTML(http.StatusOK, "products.tmpl", gin.H{
		"title":    "Products",
		"products": products,
	})
}

func SetupRouter(dbh database.DatabaseHandler) *gin.Engine {
	dbHandler = dbh

	currDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	router := gin.Default()
	router.Static("/assets", filepath.Join(currDir, "app/assets"))
	router.StaticFile("/favicon.ico", filepath.Join(currDir, "app/assets/favicon.ico"))
	router.LoadHTMLGlob(currDir + "/app/templates/*")

	router.POST("/admin/init", postInitEndpoint)

	router.GET("/product/:product_id", getProductEndpoint)
	router.GET("/products", getProductsEndpoint)
	router.GET("/checkouts", getCheckoutsEndpoint)
	router.POST("/checkout", postCheckoutEndpoint)

	return router
}
