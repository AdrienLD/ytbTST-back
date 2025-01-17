package routes

import (
	"database/sql"
	"net/http"
	"ytst-back/config"
	"ytst-back/logic"

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
	router.GET("/ytbtst/addChannel", addChannel)
	router.GET("/ytbtst/checkNewVideos", checkNewVideos)

	dbConn = db
	return router
}

func ytstResearch(c *gin.Context) {
	searchValue := c.Query("searchValue")
	if searchValue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le paramètre 'searchValue' est requis"})
		return
	}

	data, err := logic.YtstResearch(searchValue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func addChannel(c *gin.Context) {
	channelId := c.Query("channelId")
	if channelId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le paramètre 'channelId' est requis"})
		return
	}
	err := logic.AddChannel(dbConn, channelId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Chaîne ajoutée avec succès", "channel": channelId})

}

func checkNewVideos(c *gin.Context) {
	channelId := c.Query("channelId")
	if channelId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le paramètre 'channelId' est requis"})
		return
	}
	err := logic.CheckNewVideos(dbConn, channelId)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vidéos ajoutées avec succès"})
}
