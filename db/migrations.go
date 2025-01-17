package db

import (
	"database/sql"
	"fmt"
	"log"
)

func RunMigrations(db *sql.DB) error {
	createChannelsTable := `
	CREATE TABLE IF NOT EXISTS channels (
		id SERIAL PRIMARY KEY,
		channel_id VARCHAR(255) NOT NULL UNIQUE,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		thumbnail_url TEXT,
		country VARCHAR(2),
		custom_url VARCHAR(255),
		created_at TIMESTAMP,
		added_at TIMESTAMP DEFAULT NOW()
	);`

	createChannelStatsTable := `
	CREATE TABLE IF NOT EXISTS channel_stats (
		id SERIAL PRIMARY KEY,
		channel_id INT NOT NULL,
		subscribers_count INT,
		views_count BIGINT,
		videos_count INT,
		recorded_at TIMESTAMP DEFAULT NOW(),
		FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
	);`

	createVideosTable := `
	CREATE TABLE IF NOT EXISTS videos (
	 	id SERIAL PRIMARY KEY,
		video_id VARCHAR(255) NOT NULL UNIQUE, 
		is_short BOOLEAN DEFAULT FALSE,
		channel_id INT NOT NULL,
		title VARCHAR(255) NOT NULL,      
		description TEXT,      
		published_at TIMESTAMP NOT NULL,  
		thumbnail_url TEXT,
		added_at TIMESTAMP DEFAULT NOW(),
		refreshed_frequency INTERVAL DEFAULT '2 hours',
		FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
	);`

	createVideoStatsTable := `
	CREATE TABLE IF NOT EXISTS video_stats (
		id SERIAL PRIMARY KEY,
		video_id INT NOT NULL,
		views_count BIGINT,
		likes_count INT,
		comments_count INT,
		recorded_at TIMESTAMP DEFAULT NOW(),
		FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE
	);`

	if _, err := db.Exec(createChannelsTable); err != nil {
		return fmt.Errorf("erreur lors de la création de la table channels : %w", err)
	}

	if _, err := db.Exec(createChannelStatsTable); err != nil {
		return fmt.Errorf("erreur lors de la création de la table channel_stats : %w", err)
	}

	if _, err := db.Exec(createVideosTable); err != nil {
		return fmt.Errorf("erreur lors de la création de la table videos : %w", err)
	}

	if _, err := db.Exec(createVideoStatsTable); err != nil {
		return fmt.Errorf("erreur lors de la création de la table video_stats : %w", err)
	}

	log.Println("Les tables ont été créées avec succès !")
	return nil
}
