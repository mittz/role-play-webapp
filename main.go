package main

import (
	"log"

	"github.com/mittz/scaledce-role-play-series/webapp/app"
	"github.com/mittz/scaledce-role-play-series/webapp/database"
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

	router := app.SetupRouter(dbHandler)
	router.Run(":8080")
}
