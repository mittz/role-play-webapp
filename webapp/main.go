package main

import (
	"log"

	"github.com/mittz/role-play-webapp/webapp/app"
	"github.com/mittz/role-play-webapp/webapp/database"
)

func main() {
	dbHandler, err := database.NewDatabaseHandler("production")
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Consider making OpeDatabase called in each http handler
	// because this causes a deployment error when a database is not ready
	if err := dbHandler.OpenDatabase(); err != nil {
		log.Fatal(err)
	}

	// TODO: Delete InitDatabase here
	// because this causes a deployment error when a database is not ready
	if err := dbHandler.InitDatabase(); err != nil {
		log.Fatal(err)
	}

	const assetsDir = "./app/assets"
	const templatesDirMatch = "./app/templates/*"

	router := app.SetupRouter(dbHandler, assetsDir, templatesDirMatch)
	router.Run(":8080")
}
