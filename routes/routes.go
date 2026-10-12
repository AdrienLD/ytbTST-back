package routes

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"ytst-back/config"
	"ytst-back/db"
	"ytst-back/logic"
	"ytst-back/youtube"

	"github.com/gin-gonic/gin"
)

var dbConn *sql.DB

func SetupRoutes(db *sql.DB) *gin.Engine {
	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", config.WebsiteAccess)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, command, method")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	})
	router.GET("/ytbtst/research", ytstResearch)
	router.POST("/ytbtst/addChannel", keepSubscriptionAlive)
	// router.GET("/ytbtst/checkNewVideos", checkNewVideos)
	router.GET("/youtube/callback", handleYouTubeHubChallenge)
	router.POST("/youtube/callback", handleYouTubeNotification)
	router.GET("/ytbtst/channelInfo", channelInfo)
	router.GET("/ytbtst/channelStats", channelStats)
	router.GET("/ytbtst/videosFromChannel", videosFromChannel)
	router.GET("/ytbtst/videoInfo", videoInfo)
	router.GET("/ytbtst/videoStats", videoStats)
	router.GET("/ytbtst/recuperateLastFollowedChannels", recuperateLastFollowedChannels)
	router.GET("/ytbtst/recuperateLastFollowedVideos", recuperateLastFollowedVideos)

	dbConn = db
	return router
}

func ytstResearch(c *gin.Context) {
	searchValue := c.Query("searchValue")
	if searchValue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le paramètre 'searchValue' est requis"})
		return
	}

	data, err := logic.YtstResearch(dbConn, searchValue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func handleYouTubeHubChallenge(c *gin.Context) {
	mode := c.Query("hub.mode")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" {
		c.String(http.StatusOK, challenge)
		return
	}

	c.Status(http.StatusOK)
}

func handleYouTubeNotification(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}
	var notification config.Feed
	if err := xml.Unmarshal(body, &notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot bind json"})
		return
	}

	if len(notification.Entry) == 0 {
		fmt.Println("Aucune entrée reçue dans la notification YouTube")
		c.JSON(http.StatusBadRequest, gin.H{"error": "No entries found in notification"})
		return
	}

	err = logic.AddNewVideo(dbConn, notification.Entry[0].VideoId, notification.Entry[0].ChannelId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func keepSubscriptionAlive(c *gin.Context) {
	var req config.AddChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	channelId := req.ChannelID
	fmt.Println("channelId", channelId)
	err := logic.AddChannel(dbConn, channelId)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Le renouvellement est géré globalement par logic.RenewAllSubscriptions
	err = youtube.SubscribeToChannel(channelId)
	if err != nil {
		log.Printf("Erreur initiale d'abonnement: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Abonnement en cours"})
}

func channelInfo(c *gin.Context) {
	channelId := c.Query("channelId")
	if channelId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le paramètre 'channelId' est requis"})
	}

	data, err := db.ChannelInfo(dbConn, channelId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func channelStats(c *gin.Context) {
	channelId := c.Query("channelId")
	if channelId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le paramètre 'channelId' est requis"})
	}

	data, err := db.ChannelStats(dbConn, channelId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func videosFromChannel(c *gin.Context) {
	channelId := c.Query("channelId")
	if channelId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le paramètre 'channelId' est requis"})
	}

	data, err := db.VideosFromChannel(dbConn, channelId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func videoInfo(c *gin.Context) {
	videoId := c.Query("videoId")
	if videoId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le paramètre 'videoId' est requis"})
	}

	data, err := db.VideoInfo(dbConn, videoId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func videoStats(c *gin.Context) {
	videoId := c.Query("videoId")
	if videoId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le paramètre 'videoId' est requis"})
	}

	data, err := db.VideoStats(dbConn, videoId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func recuperateLastFollowedChannels(c *gin.Context) {
	data, err := db.RecuperateLastFollowedChannels(dbConn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func recuperateLastFollowedVideos(c *gin.Context) {
	data, err := db.RecuperateLastFollowedVideos(dbConn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}
