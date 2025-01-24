package config

type Config struct {
	DBUser string
	DBPass string
	DBName string
	DBHost string
	DBPort string
	DBSSL  string

	YouTubeAPIKey string
	WebsiteAccess string
}

type YouTubeChannel struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			CustomURL   string `json:"customUrl"`
			Country     string `json:"country"`
			CreatedDate string `json:"publishedAt"`
			Thumbnails  struct {
				Default struct {
					URL string `json:"url"`
				} `json:"default"`
				Medium struct {
					URL string `json:"url"`
				} `json:"medium"`
				High struct {
					URL string `json:"url"`
				} `json:"high"`
			} `json:"thumbnails"`
		} `json:"snippet"`
	} `json:"items"`
}

type YouTubeChannelStats struct {
	Items []struct {
		ID         string `json:"id"`
		Statistics struct {
			SubscribersCount string `json:"subscriberCount"`
			ViewsCount       string `json:"viewCount"`
			VideoCount       string `json:"videoCount"`
		} `json:"statistics"`
	} `json:"items"`
}

type YouTubeVideo struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			PublishedAt string `json:"publishedAt"`
			ChannelId   string `json:"channelId"`
			Thumbnails  struct {
				Default struct {
					URL string `json:"url"`
				} `json:"default"`
				Medium struct {
					URL string `json:"url"`
				} `json:"medium"`
				High struct {
					URL string `json:"url"`
				} `json:"high"`
			} `json:"thumbnails"`
		} `json:"snippet"`
	} `json:"items"`
}

type YouTubeVideoStats struct {
	Items []struct {
		ID         string `json:"id"`
		Statistics struct {
			ViewsCount   string `json:"viewCount"`
			LikeCount    string `json:"likeCount"`
			CommentCount string `json:"commentCount"`
		} `json:"statistics"`
	} `json:"items"`
}

type YouTubeVideoContentDetails struct {
	Items []struct {
		ID             string `json:"id"`
		ContentDetails struct {
			Duration string `json:"duration"`
		} `json:"contentDetails"`
	} `json:"items"`
}

type Feed struct {
	Entry []struct {
		ChannelId string `xml:"channelId" xml:"http://www.youtube.com/xml/schemas/2015 channelId"`
		VideoId   string `xml:"videoId" xml:"http://www.youtube.com/xml/schemas/2015 videoId"`
	} `xml:"entry"`
}

type Channel struct {
	ID           int    `json:"id"`
	ChannelID    string `json:"channel_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ThumbnailURL string `json:"thumbnail_url"`
	Country      string `json:"country"`
	CustomURL    string `json:"custom_url"`
	CreatedAt    string `json:"created_at"`
	AddedAt      string `json:"added_at"`
}

type ChannelStats struct {
	ID              int    `json:"id"`
	ChannelID       string `json:"channel_id"`
	SubscriberCount string `json:"subscribers_count"`
	ViewsCount      string `json:"views_count"`
	VideoCount      string `json:"video_count"`
	RecordedAt      string `json:"recorded_at"`
}

type Video struct {
	ID           int    `json:"id"`
	VideoID      string `json:"video_id"`
	IsShort      bool   `json:"is_short"`
	ChannelID    string `json:"channel_id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	PublishedAt  string `json:"published_at"`
	ThumbnailURL string `json:"thumbnail_url"`
	AddedAt      string `json:"added_at"`
	Frequency    string `json:"refreshed_frequency"`
}

type VideoStats struct {
	ID            int    `json:"id"`
	VideoID       string `json:"video_id"`
	ViewsCount    string `json:"views_count"`
	LikesCount    string `json:"likes_count"`
	CommentsCount string `json:"comments_count"`
	RecordedAt    string `json:"recorded_at"`
}

type AddChannelRequest struct {
	ChannelID string `json:"channelId"`
}
