package main

import (
	"log"
	"net/http"
	"time"
	"url_shortener/database"
	"url_shortener/services"

	"github.com/gin-gonic/gin"
)

type CreateLinkRequest struct {
	OriginalURL string `json:"original_url" binding:"required"`
	CustomCode  string `json:"custom_code"`
	ExpiresIn   *int   `json:"expires_in"`
}

func main() {
	database.Connect()

	router := gin.Default()

	router.POST("/api/links", createShortLink)

	router.GET("/api/links/:code", getLinkInfo)

	router.GET("/api/links", getAllLinks)

	router.GET("/:code", redirectToOriginal)

	log.Println("URL Shortener starting on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func createShortLink(c *gin.Context) {
	var request CreateLinkRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var expiresDuration *time.Duration
	if request.ExpiresIn != nil {
		duration := time.Duration(*request.ExpiresIn) * time.Hour
		expiresDuration = &duration
	}

	link, err := services.CreateShortLink(request.OriginalURL, request.CustomCode, expiresDuration)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	baseURL := c.Request.Host
	shortURL := "http://" + baseURL + "/" + link.ShortCode

	c.JSON(http.StatusCreated, gin.H{
		"original_url": link.OriginalURL,
		"short_code":   link.ShortCode,
		"short_url":    shortURL,
		"expires_at":   link.ExpiresAt,
		"created_at":   link.CreatedAt,
	})
}

func getLinkInfo(c *gin.Context) {
	shortCode := c.Param("code")

	link, err := services.GetLinkByShortCode(shortCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Link not found or expired"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           link.ID,
		"original_url": link.OriginalURL,
		"short_code":   link.ShortCode,
		"click_count":  link.ClickCount,
		"created_at":   link.CreatedAt,
		"expires_at":   link.ExpiresAt,
	})
}

func getAllLinks(c *gin.Context) {
	page := 1
	pageSize := 10

	links, total, err := services.GetAllLinks(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"links": links,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func redirectToOriginal(c *gin.Context) {
	shortCode := c.Param("code")

	link, err := services.GetLinkByShortCode(shortCode)
	if err != nil {
		c.String(http.StatusNotFound, "Link not found or expired")
		return
	}

	referrer := c.Request.Referer()
	userAgent := c.Request.UserAgent()
	ipAddress := c.ClientIP()

	go func() {
		if err := services.RecordClick(link, referrer, userAgent, ipAddress); err != nil {
			log.Printf("Failed to record click: %v", err)
		}
	}()

	c.Redirect(http.StatusMovedPermanently, link.OriginalURL)
}
