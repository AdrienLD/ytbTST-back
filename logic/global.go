package logic

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
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

	var channelIDs, videoIDs []string

	for _, item := range data["items"].([]interface{}) {
		itMap := item.(map[string]interface{})
		idPart := itMap["id"].(map[string]interface{})

		kind := idPart["kind"].(string)
		switch kind {
		case "youtube#channel":
			ch := idPart["channelId"].(string)
			channelIDs = append(channelIDs, ch)
		case "youtube#video":
			vid := idPart["videoId"].(string)
			videoIDs = append(videoIDs, vid)
		}
	}

	channelsMap, err := db.AreChannelsInBDD(dbConn, channelIDs)
	videosMap, err := db.AreVideosInBDD(dbConn, videoIDs)
	if err != nil {
		return nil, fmt.Errorf("Erreur lors de la recherche des chaînes en base de données : %v", err)
	}

	for _, item := range data["items"].([]interface{}) {
		itMap := item.(map[string]interface{})
		idPart := itMap["id"].(map[string]interface{})
		kind := idPart["kind"].(string)

		switch kind {
		case "youtube#channel":
			ch := idPart["channelId"].(string)
			// On ajoute un champ dans le snippet ou n’importe où
			itMap["existsInDB"] = channelsMap[ch]
		case "youtube#video":
			vid := idPart["videoId"].(string)
			itMap["existsInDB"] = videosMap[vid]
		}
	}

	return data, nil
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

	_, err = db.Query(query, channel.ID, snippet.Title, snippet.Description, bestThumbnail, snippet.Country, snippet.CustomURL, snippet.CreatedDate)
	if err != nil {
		fmt.Printf("Erreur lors de l'insertion en base de données : %v\n", err)
		return fmt.Errorf("Erreur lors de l'insertion en base de données : %v", err)
	}

	fmt.Printf("Chaîne ajoutée avec succès pour channel_id '%s' avec l'ID '%s'.\n", channelId)
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
		log.Fatalf("Erreur lors de la récupération des chaînes : %v", err)
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
	data, err := youtube.YouTubeAPIRequest("videos", map[string]string{
		"part":       "snippet",
		"id":         videoId,
		"maxResults": "1",
	})
	if err != nil {
		return fmt.Errorf("Erreur lors de l'appel à l'API YouTube pour video_id '%s': %v\n", videoId, err)
	}

	query := `SELECT id FROM channels WHERE channel_id = $1;`
	var dbChannelID int
	err = db.QueryRow(query, channelId).Scan(&dbChannelID)

	if err != nil {
		return fmt.Errorf("Erreur lors de l'appel à l'API YouTube pour video_id '%s': %v\n", videoId, err)
	}

	var videoData config.YouTubeVideo
	if err := mapToStruct(data, &videoData); err != nil {
		return fmt.Errorf("Erreur lors du traitement des données de la vidéo : %v\n", err)
	}

	var isaShort bool = isShort(videoId)

	video := videoData.Items[0]
	query = `
		INSERT INTO videos (video_id, channel_id, title, description, published_at, thumbnail_url, is_short)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
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
	if err != nil {
		fmt.Printf("Erreur lors de l'insertion en base de données : %v\n", err)
		return fmt.Errorf("Erreur lors de l'insertion des statistiques en base pour channel_id '%s': %v\n", video.ID, err)
	}

	fmt.Printf("Vidéo ajoutée avec succès pour video_id '%s' avec l'ID '%s'.\n", videoId, id)
	ScanVideoStats(db, id, videoId)

	return nil
}

func isShort(videoId string) bool {
	data, err := youtube.YouTubeAPIRequest("videos", map[string]string{
		"part":       "contentDetails",
		"id":         videoId,
		"maxResults": "1",
	})
	if err != nil {
		fmt.Printf("Erreur lors de l'appel à l'API YouTube pour video_id '%s': %v\n", videoId, err)
		return false
	}

	var videoContentDetails config.YouTubeVideoContentDetails
	if err := mapToStruct(data, &videoContentDetails); err != nil {
		fmt.Printf("Erreur lors du traitement des données de la vidéo : %v\n", err)
		return false
	}

	duration := videoContentDetails.Items[0].ContentDetails.Duration
	re := regexp.MustCompile(`PT(?:(\d+)M)?(?:(\d+)S)?`)
	matches := re.FindStringSubmatch(duration)
	if len(matches) == 0 {
		fmt.Printf("Durée non reconnue pour la vidéo ID %s : %s\n", videoId, duration)
		return false
	}

	minutesInt, secondsInt := 0, 0
	if matches[1] != "" {
		minutesInt, err = strconv.Atoi(matches[1])
		if err != nil {
			fmt.Printf("Erreur lors de la conversion des minutes : %v\n", err)
			return false
		}
	}
	if matches[2] != "" {
		secondsInt, err = strconv.Atoi(matches[2])
		if err != nil {
			fmt.Printf("Erreur lors de la conversion des secondes : %v\n", err)
			return false
		}
	}

	fmt.Printf("minutes et secondes %d %d\n", minutesInt, secondsInt)
	if minutesInt < 1 && secondsInt <= 60 {
		return true
	}
	return false
}

func refreshWithFrequency(db *sql.DB, frequency time.Duration) {
	interval := fmt.Sprintf("%02d:%02d:%02d", int(frequency.Hours()), int(frequency.Minutes())%60, int(frequency.Seconds())%60)
	query := `
			SELECT id, video_id FROM videos
			WHERE refreshed_frequency = $1;
			`

	rows, err := db.Query(query, interval)
	if err != nil {
		log.Fatalf("Erreur lors de la récupération des vidéos : %v", err)
	}
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
