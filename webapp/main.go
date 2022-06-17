package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"

	"github.com/mittz/role-play-webapp/webapp/app"
	"github.com/mittz/role-play-webapp/webapp/database"
	"github.com/mittz/role-play-webapp/webapp/utils"
)

func main() {
	dbInfo := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable",
		utils.GetEnvDBHostname(),
		utils.GetEnvDBPort(),
		utils.GetEnvDBUsername(),
		utils.GetEnvDBName(),
		utils.GetEnvDBPassword(),
	)

	db, err := sql.Open("postgres", dbInfo)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	dbHandler, err := database.NewDatabaseHandler("production", db)
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
