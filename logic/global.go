package logic

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"ytst-back/config"
	"ytst-back/db"
	"ytst-back/youtube"
)

func PeriodicallyCalledRoutes(db *sql.DB) {
	fmt.Println("Appels périodiques des routes...")
	callRoutePeriodically(updateAllChannelStats, 24*time.Hour, db)
	//callRoutePeriodically(autoCheckNewVideos, 2*time.Hour, db)
	callRoutePeriodically(refreshWithFrequency, 2*time.Hour, db)

	// Les abonnements PubSubHubbub ne survivent pas à un redémarrage : on les renouvelle tous au démarrage, puis avant chaque expiration du bail
	go RenewAllSubscriptions(db, youtube.RenewInterval)
	callRoutePeriodically(RenewAllSubscriptions, youtube.RenewInterval, db)
}

func RenewAllSubscriptions(db *sql.DB, _ time.Duration) {
	rows, err := db.Query(`SELECT channel_id FROM channels`)
	if err != nil {
		log.Printf("Erreur lors de la récupération des chaînes pour le renouvellement : %v", err)
		return
	}

	var channelIDs []string
	for rows.Next() {
		var channelID string
		if err := rows.Scan(&channelID); err != nil {
			log.Printf("Erreur lors de la lecture de la chaîne : %v", err)
			continue
		}
		channelIDs = append(channelIDs, channelID)
	}
	rows.Close()

	for _, channelID := range channelIDs {
		if err := youtube.SubscribeToChannel(channelID); err != nil {
			log.Printf("Erreur de renouvellement d'abonnement pour la chaîne %s : %v", channelID, err)
		} else {
			log.Printf("Renouvellement d'abonnement OK pour la chaîne : %s", channelID)
		}
	}
}

func mapToStruct(data interface{}, result interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, result)
}

func YtstResearch(dbConn *sql.DB, searchValue string) (map[string]interface{}, error) {

	data, err := youtube.YouTubeAPIRequest("search", map[string]string{
		"part": "snippet",
		"q":    searchValue,
	})

	if err != nil {
		return nil, fmt.Errorf("Erreur lors de l'appel à l'API YouTube pour searchValue '%s': %v", searchValue, err)
	}

	items, _ := data["items"].([]interface{})

	var channelIDs, videoIDs []string
	for _, item := range items {
		itMap, kind, id := searchItemID(item)
		if itMap == nil {
			continue
		}
		switch kind {
		case "youtube#channel":
			channelIDs = append(channelIDs, id)
		case "youtube#video":
			videoIDs = append(videoIDs, id)
		}
	}

	channelsMap, err := db.AreChannelsInBDD(dbConn, channelIDs)
	if err != nil {
		return nil, fmt.Errorf("Erreur lors de la recherche des chaînes en base de données : %v", err)
	}
	videosMap, err := db.AreVideosInBDD(dbConn, videoIDs)
	if err != nil {
		return nil, fmt.Errorf("Erreur lors de la recherche des vidéos en base de données : %v", err)
	}

	for _, item := range items {
		itMap, kind, id := searchItemID(item)
		if itMap == nil {
			continue
		}
		switch kind {
		case "youtube#channel":
			itMap["existsInDB"] = channelsMap[id]
		case "youtube#video":
			itMap["existsInDB"] = videosMap[id]
		}
	}

	return data, nil
}

// Extrait le type et l'ID d'un résultat de recherche YouTube, sans paniquer si le format est inattendu
func searchItemID(item interface{}) (map[string]interface{}, string, string) {
	itMap, ok := item.(map[string]interface{})
	if !ok {
		return nil, "", ""
	}
	idPart, ok := itMap["id"].(map[string]interface{})
	if !ok {
		return nil, "", ""
	}
	kind, _ := idPart["kind"].(string)
	switch kind {
	case "youtube#channel":
		id, _ := idPart["channelId"].(string)
		return itMap, kind, id
	case "youtube#video":
		id, _ := idPart["videoId"].(string)
		return itMap, kind, id
	}
	return itMap, kind, ""
}

func AddChannel(db *sql.DB, channelId string) error {
	data, err := youtube.YouTubeAPIRequest("channels", map[string]string{
		"part": "snippet",
		"id":   channelId,
	})

	if err != nil {
		fmt.Printf("Erreur lors de l'appel à l'API YouTube pour channel_id '%s': %v\n", channelId, err)
		return fmt.Errorf("Erreur lors de l'appel à l'API YouTube pour channel_id '%s': %v", channelId, err)
	}

	var channelData config.YouTubeChannel
	if err := mapToStruct(data, &channelData); err != nil {
		fmt.Printf("Erreur lors du traitement des données de la chaîne : %v\n", err)
		return fmt.Errorf("Erreur lors du traitement des données de la chaîne : %v", err)
	}

	if len(channelData.Items) == 0 {
		return fmt.Errorf("Aucune chaîne YouTube trouvée pour channel_id '%s'", channelId)
	}
	channel := channelData.Items[0]
	snippet := channel.Snippet

	query := `
		INSERT INTO channels (channel_id, name, description, thumbnail_url, country, custom_url, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id;
	`

	var bestThumbnail string
	if channel.Snippet.Thumbnails.High.URL != "" {
		bestThumbnail = channel.Snippet.Thumbnails.High.URL
	} else if channel.Snippet.Thumbnails.Medium.URL != "" {
		bestThumbnail = channel.Snippet.Thumbnails.Medium.URL
	} else {
		bestThumbnail = channel.Snippet.Thumbnails.Default.URL
	}

	var id int
	err = db.QueryRow(query, channel.ID, snippet.Title, snippet.Description, bestThumbnail, snippet.Country, snippet.CustomURL, snippet.CreatedDate).Scan(&id)
	if err != nil {
		fmt.Printf("Erreur lors de l'insertion en base de données : %v\n", err)
		return fmt.Errorf("Erreur lors de l'insertion en base de données : %v", err)
	}

	fmt.Printf("Chaîne ajoutée avec succès pour channel_id '%s' avec l'ID '%d'.\n", channelId, id)
	refreshChannelStats(db, channel.ID)

	return nil
}

func refreshChannelStats(db *sql.DB, channelId string) {
	if channelId == "" {
		fmt.Println("Le paramètre 'channelId' est requis.")
		return
	}

	dbChannelID, err := recuperateChannelFromDB(db, channelId)
	if err != nil {
		fmt.Printf("Erreur lors de la récupération de l'ID de la chaîne : %v\n", err)
		return
	}

	data, err := youtube.YouTubeAPIRequest("channels", map[string]string{
		"part": "statistics",
		"id":   channelId,
	})
	if err != nil {
		fmt.Printf("Erreur lors de l'appel à l'API YouTube pour channel_id '%s': %v\n", channelId, err)
		return
	}

	var channelData config.YouTubeChannelStats
	if err := mapToStruct(data, &channelData); err != nil {
		fmt.Printf("Erreur lors du traitement des données de la chaîne : %v\n", err)
		return
	}

	if len(channelData.Items) == 0 {
		fmt.Printf("Aucune donnée trouvée pour la chaîne avec channel_id '%s'.\n", channelId)
		return
	}

	channel := channelData.Items[0]
	statistics := channel.Statistics

	query := `
		INSERT INTO channel_stats (channel_id, subscribers_count, views_count, videos_count)
		VALUES ($1, $2, $3, $4);
	`

	_, err = db.Exec(query, dbChannelID, statistics.SubscribersCount, statistics.ViewsCount, statistics.VideoCount)
	if err != nil {
		fmt.Printf("Erreur lors de l'insertion des statistiques en base pour channel_id '%s': %v\n", channelId, err)
		return
	}

	fmt.Printf("Statistiques mises à jour avec succès pour channel_id '%s'.\n", channelId)
}

func updateAllChannelStats(db *sql.DB, _ time.Duration) {
	fmt.Println("Mise à jour des statistiques de toutes les chaînes...")
	rows, err := db.Query(`SELECT channel_id FROM channels`)
	if err != nil {
		log.Printf("Erreur lors de la récupération des chaînes : %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var channelID string
		if err := rows.Scan(&channelID); err != nil {
			log.Printf("Erreur lors de la récupération de la chaîne : %v", err)
			continue
		}

		refreshChannelStats(db, channelID)
	}
}

func callRoutePeriodically(task func(*sql.DB, time.Duration), interval time.Duration, dbConn *sql.DB) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			task(dbConn, interval)
		}
	}()
}

func recuperateChannelFromDB(db *sql.DB, channelId string) (int, error) {

	var dbChannelID int
	err := db.QueryRow(`SELECT id FROM channels WHERE channel_id = $1`, channelId).Scan(&dbChannelID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("La chaîne avec channel_id '%s' n'existe pas dans la base de données.\n", channelId)
		} else {
			fmt.Printf("Erreur lors de la récupération de l'ID de la chaîne : %v\n", err)
		}
		return 0, err
	}

	return dbChannelID, nil
}

func AddNewVideo(db *sql.DB, videoId string, channelId string) error {
	// Vérifications en base d'abord : évite de consommer du quota pour une vidéo déjà connue ou une chaîne non suivie
	var dbChannelID int
	err := db.QueryRow(`SELECT id FROM channels WHERE channel_id = $1;`, channelId).Scan(&dbChannelID)
	if err == sql.ErrNoRows {
		fmt.Printf("Notification ignorée : la chaîne '%s' n'est pas suivie.\n", channelId)
		return nil
	}
	if err != nil {
		return fmt.Errorf("Erreur lors de la récupération de la chaîne '%s' : %v", channelId, err)
	}

	var exists bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM videos WHERE video_id = $1);`, videoId).Scan(&exists); err != nil {
		return fmt.Errorf("Erreur lors de la vérification de la vidéo '%s' : %v", videoId, err)
	}
	if exists {
		// YouTube renvoie une notification à chaque modification de la vidéo (titre, miniature...)
		fmt.Printf("Notification ignorée : la vidéo '%s' est déjà enregistrée.\n", videoId)
		return nil
	}

	data, err := youtube.YouTubeAPIRequest("videos", map[string]string{
		"part": "snippet",
		"id":   videoId,
	})
	if err != nil {
		return fmt.Errorf("Erreur lors de l'appel à l'API YouTube pour video_id '%s': %v", videoId, err)
	}

	var videoData config.YouTubeVideo
	if err := mapToStruct(data, &videoData); err != nil {
		return fmt.Errorf("Erreur lors du traitement des données de la vidéo : %v", err)
	}

	if len(videoData.Items) == 0 {
		// Vidéo privée ou supprimée entre la notification et l'appel
		fmt.Printf("Aucune donnée trouvée pour la vidéo avec video_id '%s'.\n", videoId)
		return nil
	}
	video := videoData.Items[0]
	isaShort, err := youtube.IsShort(videoId)
	if err != nil {
		fmt.Printf("Impossible de déterminer si la vidéo '%s' est un Short, considérée comme vidéo classique : %v\n", videoId, err)
	}

	query := `
		INSERT INTO videos (video_id, channel_id, title, description, published_at, thumbnail_url, is_short)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (video_id) DO NOTHING
		RETURNING id;
	`

	var bestThumbnail string
	if video.Snippet.Thumbnails.High.URL != "" {
		bestThumbnail = video.Snippet.Thumbnails.High.URL
	} else if video.Snippet.Thumbnails.Medium.URL != "" {
		bestThumbnail = video.Snippet.Thumbnails.Medium.URL
	} else {
		bestThumbnail = video.Snippet.Thumbnails.Default.URL
	}

	var id string
	err = db.QueryRow(query, video.ID, dbChannelID, video.Snippet.Title, video.Snippet.Description, video.Snippet.PublishedAt, bestThumbnail, isaShort).Scan(&id)
	if err == sql.ErrNoRows {
		// Insérée entre-temps par une notification concurrente
		return nil
	}
	if err != nil {
		fmt.Printf("Erreur lors de l'insertion en base de données : %v\n", err)
		return fmt.Errorf("Erreur lors de l'insertion de la vidéo '%s' en base : %v", video.ID, err)
	}

	fmt.Printf("Vidéo ajoutée avec succès pour video_id '%s' avec l'ID '%s'.\n", videoId, id)
	ScanVideoStats(db, id, videoId)

	return nil
}

func refreshWithFrequency(db *sql.DB, frequency time.Duration) {
	interval := fmt.Sprintf("%02d:%02d:%02d", int(frequency.Hours()), int(frequency.Minutes())%60, int(frequency.Seconds())%60)
	query := `
			SELECT id, video_id FROM videos
			WHERE refreshed_frequency = $1;
			`

	rows, err := db.Query(query, interval)
	if err != nil {
		log.Printf("Erreur lors de la récupération des vidéos : %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var videoID string
		if err := rows.Scan(&id, &videoID); err != nil {
			log.Printf("Erreur lors de la lecture des résultats : %v", err)
			continue
		}
		ScanVideoStats(db, id, videoID)
	}
}

func ScanVideoStats(db *sql.DB, id string, videoId string) {
	fmt.Printf("Mise à jour des statistiques pour video_id '%s'...\n", videoId)
	data, err := youtube.YouTubeAPIRequest("videos", map[string]string{
		"part": "statistics",
		"id":   videoId,
	})

	if err != nil {
		fmt.Printf("Erreur lors de l'appel à l'API YouTube pour video_id '%s': %v\n", videoId, err)
		return
	}

	var videoData config.YouTubeVideoStats
	if err := mapToStruct(data, &videoData); err != nil {
		fmt.Printf("Erreur lors du traitement des données de la vidéo : %v\n", err)
		return
	}

	if len(videoData.Items) == 0 {
		fmt.Printf("Aucune donnée trouvée pour la vidéo avec video_id '%s'.\n", videoId)
		return
	}

	video := videoData.Items[0]
	statistics := video.Statistics

	query := `
		INSERT INTO video_stats (video_id, views_count, likes_count, comments_count)
		VALUES ($1, $2, $3, $4);
	`
	_, err = db.Exec(query, id, statistics.ViewsCount, statistics.LikeCount, statistics.CommentCount)
	if err != nil {
		fmt.Printf("Erreur lors de l'insertion des statistiques en base pour video_id '%s': %v\n", videoId, err)
		return
	}

	fmt.Printf("Statistiques mises à jour avec succès pour video_id '%s'.\n", videoId)
}
