package main

import (
	"log"
	"net/http"
	"strconv"
	"time"
	"url_shortener/auth"
	"url_shortener/database"
	"url_shortener/handlers"
	"url_shortener/models"
	"url_shortener/services"

	"github.com/gin-gonic/gin"
)

type CreateLinkRequest struct {
	OriginalURL string   `json:"original_url" binding:"required"`
	CustomCode  string   `json:"custom_code"`
	ExpiresIn   *int     `json:"expires_in"`
	Tags        []string `json:"tags"`
}

type UpdateLinkRequest struct {
	OriginalURL string `json:"original_url"`
	CustomCode  string `json:"custom_code"`
	ExpiresIn   *int   `json:"expires_in"`
}

type TagRequest struct {
	Name string `json:"name" binding:"required"`
}

func main() {
	database.Connect()

	database.DB.AutoMigrate(&models.User{}, &models.Link{}, &models.ClickStat{}, &models.Tag{}, &models.LinkTag{})

	router := gin.Default()

	router.POST("/api/register", handlers.Register)
	router.POST("/api/login", handlers.Login)
	router.GET("/:code", redirectToOriginal)

	api := router.Group("/api")
	api.Use(auth.AuthMiddleware())
	{
		api.POST("/links", createShortLink)
		api.GET("/links/:code", getLinkInfo)
		api.GET("/links", getAllLinks)
		api.PUT("/links/:code", updateLink)
		api.DELETE("/links/:code", deleteLink)

		api.GET("/links/:code/stats", getLinkStats)
		api.GET("/user/stats", getUserStats)

		api.GET("/user/profile", getUserProfile)
		api.PUT("/user/profile", updateUserProfile)

		api.POST("/tags", createTag)
		api.GET("/tags", getAllTags)
		api.GET("/tags/:name/links", getLinksByTag)

		api.POST("/links/:code/tags", addTagToLink)
		api.DELETE("/links/:code/tags/:tag_id", removeTagFromLink)
		api.GET("/dashboard", getDashboardData)
	}

	log.Println("URL Shortener starting on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func createShortLink(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

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

	link, err := services.CreateShortLink(request.OriginalURL, request.CustomCode, expiresDuration, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(request.Tags) > 0 {
		for _, tagName := range request.Tags {
			var tag models.Tag
			result := database.DB.Where("name = ?", tagName).FirstOrCreate(&tag, models.Tag{Name: tagName})
			if result.Error != nil {
				log.Printf("Error creating tag: %v", result.Error)
				continue
			}

			linkTag := models.LinkTag{
				LinkID: link.ID,
				TagID:  tag.ID,
			}
			database.DB.Create(&linkTag)
		}
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
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	shortCode := c.Param("code")

	link, err := services.GetLinkByShortCode(shortCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Link not found or expired"})
		return
	}

	if link.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this link"})
		return
	}

	var linkTags []models.Tag
	database.DB.Table("tags").
		Joins("JOIN link_tags ON link_tags.tag_id = tags.id").
		Where("link_tags.link_id = ?", link.ID).
		Find(&linkTags)

	c.JSON(http.StatusOK, gin.H{
		"id":           link.ID,
		"original_url": link.OriginalURL,
		"short_code":   link.ShortCode,
		"click_count":  link.ClickCount,
		"created_at":   link.CreatedAt,
		"expires_at":   link.ExpiresAt,
		"tags":         linkTags,
	})
}

func getAllLinks(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	links, total, err := services.GetUserLinks(userID, page, pageSize)
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

func updateLink(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	linkID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid link ID"})
		return
	}

	var request UpdateLinkRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var expiresDuration *time.Duration
	if request.ExpiresIn != nil {
		duration := time.Duration(*request.ExpiresIn) * time.Hour
		expiresDuration = &duration
	}

	link, err := services.UpdateLink(uint(linkID), userID, request.OriginalURL, request.CustomCode, expiresDuration)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	baseURL := c.Request.Host
	shortURL := "http://" + baseURL + "/" + link.ShortCode

	c.JSON(http.StatusOK, gin.H{
		"id":           link.ID,
		"original_url": link.OriginalURL,
		"short_code":   link.ShortCode,
		"short_url":    shortURL,
		"expires_at":   link.ExpiresAt,
		"updated_at":   time.Now(),
	})
}

func deleteLink(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	linkID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid link ID"})
		return
	}

	if err := services.DeleteLink(uint(linkID), userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Link deleted successfully"})
}

func getLinkStats(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	linkID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid link ID"})
		return
	}

	clickStats, err := services.GetClickStats(uint(linkID), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"link_id":      linkID,
		"click_stats":  clickStats,
		"total_clicks": len(clickStats),
	})
}

func getUserStats(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var totalLinks int64
	database.DB.Model(&models.Link{}).Where("user_id = ?", userID).Count(&totalLinks)

	var totalClicks int64
	database.DB.Model(&models.Link{}).Where("user_id = ?", userID).Select("SUM(click_count)").Row().Scan(&totalClicks)

	var popularLinks []models.Link
	database.DB.Where("user_id = ?", userID).Order("click_count desc").Limit(5).Find(&popularLinks)

	c.JSON(http.StatusOK, gin.H{
		"total_links":   totalLinks,
		"total_clicks":  totalClicks,
		"popular_links": popularLinks,
	})
}

func getUserProfile(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	user, err := services.GetUserProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"created_at": user.CreatedAt,
	})
}

type UpdateProfileRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func updateUserProfile(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request UpdateProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
		return
	}

	if request.Email != "" {
		var existingUser models.User
		if database.DB.Where("email = ? AND id != ?", request.Email, userID).First(&existingUser).Error == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already in use"})
			return
		}
		user.Email = request.Email
	}

	if request.Password != "" {
		user.Password = request.Password
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"updated_at": time.Now(),
	})
}

func createTag(c *gin.Context) {
	_, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request TagRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tag models.Tag
	result := database.DB.Where("name = ?", request.Name).FirstOrCreate(&tag, models.Tag{Name: request.Name})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":   tag.ID,
		"name": tag.Name,
	})
}

func getAllTags(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var tags []models.Tag
	database.DB.Distinct("tags.*").
		Joins("JOIN link_tags ON link_tags.tag_id = tags.id").
		Joins("JOIN links ON links.id = link_tags.link_id").
		Where("links.user_id = ?", userID).
		Find(&tags)

	c.JSON(http.StatusOK, gin.H{
		"tags": tags,
	})
}

func getLinksByTag(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	tagName := c.Param("name")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	links, total, err := services.GetLinksByTag(tagName, userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tag":   tagName,
		"links": links,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func addTagToLink(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	linkID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid link ID"})
		return
	}

	var link models.Link
	if err := database.DB.Where("id = ? AND user_id = ?", linkID, userID).First(&link).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Link not found or you don't have permission"})
		return
	}

	var request TagRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tag models.Tag
	result := database.DB.Where("name = ?", request.Name).FirstOrCreate(&tag, models.Tag{Name: request.Name})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag"})
		return
	}

	var existingLinkTag models.LinkTag
	result = database.DB.Where("link_id = ? AND tag_id = ?", linkID, tag.ID).First(&existingLinkTag)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Tag already added to this link"})
		return
	}

	linkTag := models.LinkTag{
		LinkID: uint(linkID),
		TagID:  tag.ID,
	}
	if err := database.DB.Create(&linkTag).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add tag to link"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"link_id": linkID,
		"tag":     tag,
		"message": "Tag successfully added to link",
	})
}

func removeTagFromLink(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	linkID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid link ID"})
		return
	}

	tagID, err := strconv.ParseUint(c.Param("tag_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	var link models.Link
	if err := database.DB.Where("id = ? AND user_id = ?", linkID, userID).First(&link).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Link not found or you don't have permission"})
		return
	}

	result := database.DB.Where("link_id = ? AND tag_id = ?", linkID, tagID).Delete(&models.LinkTag{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found for this link"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tag successfully removed from link",
	})
}

func getDashboardData(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var totalLinks int64
	database.DB.Model(&models.Link{}).Where("user_id = ?", userID).Count(&totalLinks)

	var totalClicks int64
	database.DB.Model(&models.Link{}).Where("user_id = ?", userID).Select("SUM(click_count)").Row().Scan(&totalClicks)

	var popularLinks []models.Link
	database.DB.Where("user_id = ?", userID).Order("click_count desc").Limit(5).Find(&popularLinks)

	var recentLinks []models.Link
	database.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(5).Find(&recentLinks)

	var expiringLinks []models.Link
	expiryThreshold := time.Now().Add(time.Hour * 24 * 7)
	database.DB.Where("user_id = ? AND expires_at IS NOT NULL AND expires_at <= ?", userID, expiryThreshold).
		Order("expires_at asc").Limit(5).Find(&expiringLinks)

	c.JSON(http.StatusOK, gin.H{
		"total_links":    totalLinks,
		"total_clicks":   totalClicks,
		"popular_links":  popularLinks,
		"recent_links":   recentLinks,
		"expiring_links": expiringLinks,
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
