package handlers

import (
	"regexp"

	"github.com/gin-gonic/gin"
)

func SetupRouter(re *regexp.Regexp) *gin.Engine {
	router := gin.Default()

	router.Static("/static", "./static")
	router.LoadHTMLGlob("templates/*")

	tenderGroup := router.Group("/tender")
	{
		// HTML страница
		tenderGroup.GET("/", func(c *gin.Context) {
			c.HTML(200, "index.html", nil)
		})

		// API endpoints
		tenderGroup.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "OK", "service": "tender"})
		})

		tenderGroup.POST("/searchTenders", searchTenders(re))
		tenderGroup.GET("/download", func(c *gin.Context) {
			filename := c.Query("filename")
			if filename == "" {
				filename = "Закупки.xlsx"
			}

			c.File(filename)
		})

	}
	return router
}
