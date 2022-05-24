package main

import (
	"log"

	"github.com/mittz/role-play-webapp/webapp/app"
	"github.com/mittz/role-play-webapp/webapp/database"
)

func main() {
	dbConn, err := database.InitializeProdDBConn()
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	dbHandler, err := database.NewDatabaseHandler("production", dbConn)
	if err != nil {
		log.Fatal(err)
	}

	if err := dbHandler.InitDatabase(); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}

	const assetsDir = "./app/assets"
	const templatesDirMatch = "./app/templates/*"

	router := app.SetupRouter(dbHandler, assetsDir, templatesDirMatch)
	router.Run(":8080")
}
