package db

import (
	"database/sql"
	"fmt"
	"ytst-back/config"

	_ "github.com/lib/pq"
)

func Connect(cfg *config.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBUser, cfg.DBPass, cfg.DBName, cfg.DBSSL,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("impossible de se connecter à la base de données : %v", err)
	}

	return db, nil
}
