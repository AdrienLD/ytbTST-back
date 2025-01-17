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
	"ytst-back/youtube"
)

func PeriodicallyCalledRoutes(db *sql.DB) {
	callRoutePeriodically(refreshWithFrequency, 2*time.Hour, db)
	callRoutePeriodically(updateAllChannelStats, 15*time.Second, db)
}

func mapToStruct(data interface{}, result interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, result)
}

func YtstResearch(searchValue string) (map[string]interface{}, error) {

	data, err := youtube.YouTubeAPIRequest("search", map[string]string{
		"part": "snippet",
		"q":    searchValue,
	})

	if err != nil {
		return nil, fmt.Errorf("Erreur lors de l'appel à l'API YouTube pour searchValue '%s': %v", searchValue, err)
	}

	return data, nil
}

func AddChannel(db *sql.DB, channelId string) error {
	data, err := youtube.YouTubeAPIRequest("channels", map[string]string{
		"part": "snippet",
		"id":   channelId,
	})

	if err != nil {
		return fmt.Errorf("Erreur lors de l'appel à l'API YouTube pour channel_id '%s': %v", channelId, err)
	}

	var channelData config.YouTubeChannel
	if err := mapToStruct(data, &channelData); err != nil {
		return fmt.Errorf("Erreur lors du traitement des données de la chaîne : %v", err)
	}

	channel := channelData.Items[0]
	snippet := channel.Snippet

	query := `
		INSERT INTO channels (channel_id, name, description, thumbnail_url, country, custom_url, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7);
	`

	if _, err := db.Exec(query, channel.ID, snippet.Title, snippet.Description, snippet.Thumbnails.Default.URL, snippet.Country, snippet.CustomURL, snippet.CreatedDate); err != nil {
		return fmt.Errorf("Erreur lors de l'insertion en base de données : %v", err)
	}

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

func CheckNewVideos(db *sql.DB, channelId string) error {
	channelDBId, err := recuperateChannelFromDB(db, channelId)
	if err != nil {
		return fmt.Errorf("Erreur lors de la récupération de l'ID de la chaîne : %v\n", err)
	}

	data, err := youtube.YouTubeAPIRequest("search", map[string]string{
		"part":       "snippet",
		"channelId":  channelId,
		"maxResults": "5",
		"order":      "date",
		"type":       "video",
	})

	if err != nil {
		return fmt.Errorf("Erreur lors de l'appel à l'API YouTube pour channel_id '%s': %v\n", channelId, err)
	}

	for _, item := range data["items"].([]interface{}) {
		snippet := item.(map[string]interface{})["snippet"].(map[string]interface{})
		id := item.(map[string]interface{})["id"].(map[string]interface{})
		if snippet["publishedAt"].(string) > "2025-01-10T00:00:00Z" {
			err := addNewVideo(db, id["videoId"].(string), channelDBId)
			if err != nil {
				return fmt.Errorf("Erreur lors de l'ajout de la vidéo : %v\n", err)
			}
		}
	}

	return nil
}

func addNewVideo(db *sql.DB, videoId string, channelId int) error {
	data, err := youtube.YouTubeAPIRequest("videos", map[string]string{
		"part":       "snippet",
		"id":         videoId,
		"maxResults": "1",
	})
	if err != nil {
		return fmt.Errorf("Erreur lors de l'appel à l'API YouTube pour video_id '%s': %v\n", videoId, err)
	}

	var videoData config.YouTubeVideo
	if err := mapToStruct(data, &videoData); err != nil {
		return fmt.Errorf("Erreur lors du traitement des données de la vidéo : %v\n", err)
	}

	var isaShort bool = isShort(videoId)

	video := videoData.Items[0]
	query := `
		INSERT INTO videos (video_id, channel_id, title, description, published_at, thumbnail_url, is_short)
		VALUES ($1, $2, $3, $4, $5, $6, $7);
	`

	_, err = db.Exec(query, video.ID, channelId, video.Snippet.Title, video.Snippet.Description, video.Snippet.PublishedAt, video.Snippet.Thumbnails.Default.URL, isaShort)
	if err != nil {
		return fmt.Errorf("Erreur lors de l'insertion des statistiques en base pour channel_id '%s': %v\n", video.ID, err)
	}

	return nil
}

func isShort(videoId string) bool {
	data, _ := youtube.YouTubeAPIRequest("videos", map[string]string{
		"part":       "contentDetails",
		"id":         videoId,
		"maxResults": "1",
	})

	var videoContentDetails config.YouTubeVideoContentDetails
	if err := mapToStruct(data, &videoContentDetails); err != nil {
		fmt.Printf("Erreur lors du traitement des données de la vidéo : %v\n", err)
		return false
	}

	duration := videoContentDetails.Items[0].ContentDetails.Duration
	re := regexp.MustCompile(`PT(?P<minutes>\d+)M(?P<seconds>\d+)S`)
	matches := re.FindStringSubmatch(duration)
	minutesInt, err := strconv.Atoi(matches[1])
	if err != nil || len(matches) < 2 {
		return false
	}
	if minutesInt < 1 {
		return false
	} else {
		return true
	}

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
