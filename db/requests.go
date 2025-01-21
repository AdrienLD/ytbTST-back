package db

import (
	"database/sql"
	"fmt"
	"ytst-back/config"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

func AreChannelsInBDD(db *sql.DB, channelIDs []string) (map[string]bool, error) {
	rows, err := db.Query("SELECT channel_id FROM channels WHERE channel_id = ANY($1)", pq.StringArray(channelIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inDB := make(map[string]bool, len(channelIDs))
	for rows.Next() {
		var chID string
		if err := rows.Scan(&chID); err == nil {
			inDB[chID] = true
		}
	}
	return inDB, nil
}

func AreVideosInBDD(db *sql.DB, videosIDs []string) (map[string]bool, error) {
	rows, err := db.Query("SELECT video_id FROM videos WHERE video_id = ANY($1)", pq.StringArray(videosIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inDB := make(map[string]bool, len(videosIDs))
	for rows.Next() {
		var vID string
		if err := rows.Scan(&vID); err == nil {
			inDB[vID] = true
		}
	}
	return inDB, nil
}

func ChannelInfo(db *sql.DB, channelID string) (config.Channel, error) {
	var channel config.Channel
	err := db.QueryRow("SELECT * FROM channels WHERE channel_id = $1", channelID).Scan(
		&channel.ID,
		&channel.ChannelID,
		&channel.Name,
		&channel.Description,
		&channel.ThumbnailURL,
		&channel.Country,
		&channel.CustomURL,
		&channel.CreatedAt,
		&channel.AddedAt,
	)
	if err != nil {
		return channel, err
	}
	return channel, nil
}

func ChannelStats(db *sql.DB, channelID string) ([]config.ChannelStats, error) {

	var id int
	err := db.QueryRow("SELECT id FROM channels WHERE channel_id = $1", channelID).Scan(&id)
	if err != nil {
		return nil, err
	}

	fmt.Println("id", id)
	var statsList []config.ChannelStats
	rows, err := db.Query("SELECT * FROM channel_stats WHERE channel_id = $1", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fmt.Println("rows", rows)

	for rows.Next() {
		var stats config.ChannelStats
		if err := rows.Scan(
			&stats.ID,
			&stats.ChannelID,
			&stats.SubscriberCount,
			&stats.ViewsCount,
			&stats.VideoCount,
			&stats.RecordedAt,
		); err != nil {
			return nil, err
		}
		statsList = append(statsList, stats)

	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return statsList, nil
}
