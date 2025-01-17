package main

import (
	"database/sql"
	"log"

	"ytst-back/config"
	"ytst-back/db"
	"ytst-back/logic"
	"ytst-back/routes"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var websiteAccess string = "https://ytst.flgr.fr"

var dbConn *sql.DB

func init() {
	if err := godotenv.Load(); err != nil {
		panic("Erreur lors du chargement du fichier .env")
	}
}

func main() {

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Erreur lors du chargement de la configuration : %v", err)
	}

	dbConn, err = db.Connect(cfg)
	if err != nil {
		log.Fatalf("Erreur lors de la connexion à la base de données : %v", err)
	}
	defer dbConn.Close()

	if err := db.RunMigrations(dbConn); err != nil {
		log.Fatalf("Erreur lors de la création des tables : %v", err)
	}

	router := routes.SetupRoutes(dbConn)
	// router.GET("/ytbtst/refreshChannelStats", refreshChannelStats)

	router.Run(":4000")

	logic.PeriodicallyCalledRoutes(dbConn)
}
