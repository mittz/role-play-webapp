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
	if err := dbHandler.ReadProperties("database.json"); err != nil {
		log.Fatal(err)
	}
	if err := dbHandler.OpenDatabase(); err != nil {
		log.Fatal(err)
	}
	if err := dbHandler.InitDatabase(); err != nil {
		log.Fatal(err)
	}

	const assetsDir = "./app/assets"
	const templatesDirMatch = "./app/templates/*"

	router := app.SetupRouter(dbHandler, assetsDir, templatesDirMatch)
	router.Run(":8080")
}
