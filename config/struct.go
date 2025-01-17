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
