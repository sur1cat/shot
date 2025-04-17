package services

import (
	"crypto/rand"
	"errors"
	"math/big"
	"time"
	"url_shortener/database"
	"url_shortener/models"

	"gorm.io/gorm"
)

const (
	charset    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	codeLength = 6
)

func CreateShortLink(originalURL string, customCode string, expiresIn *time.Duration, userID uint) (*models.Link, error) {
	if originalURL == "" {
		return nil, errors.New("original URL cannot be empty")
	}

	shortCode := customCode
	if shortCode == "" {
		var err error
		shortCode, err = generateShortCode()
		if err != nil {
			return nil, err
		}
	} else {
		var existingLink models.Link
		result := database.DB.Where("short_code = ?", shortCode).First(&existingLink)
		if result.Error == nil {
			return nil, errors.New("custom short code already exists")
		} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, result.Error
		}
	}

	link := models.Link{
		UserID:      userID,
		OriginalURL: originalURL,
		ShortCode:   shortCode,
		CreatedAt:   time.Now(),
	}

	if expiresIn != nil {
		expiresAt := time.Now().Add(*expiresIn)
		link.ExpiresAt = &expiresAt
	}

	result := database.DB.Create(&link)
	if result.Error != nil {
		return nil, result.Error
	}

	return &link, nil
}

func GetLinkByShortCode(shortCode string) (*models.Link, error) {
	var link models.Link
	result := database.DB.Where("short_code = ?", shortCode).First(&link)
	if result.Error != nil {
		return nil, result.Error
	}

	if link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("link has expired")
	}

	return &link, nil
}

func RecordClick(link *models.Link, referrer, userAgent, ipAddress string) error {
	result := database.DB.Model(link).UpdateColumn("click_count", gorm.Expr("click_count + ?", 1))
	if result.Error != nil {
		return result.Error
	}

	clickStat := models.ClickStat{
		LinkID:      link.ID,
		ClickedAt:   time.Now(),
		ReferrerURL: referrer,
		UserAgent:   userAgent,
		IPAddress:   ipAddress,
	}

	result = database.DB.Create(&clickStat)
	return result.Error
}

func GetAllLinks(page, pageSize int, userID uint) ([]models.Link, int64, error) {
	var links []models.Link
	var total int64

	query := database.DB.Model(&models.Link{})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	query.Count(&total)

	result := query.Limit(pageSize).Offset((page - 1) * pageSize).Order("created_at desc").Find(&links)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	return links, total, nil
}

func GetUserLinks(userID uint, page, pageSize int) ([]models.Link, int64, error) {
	return GetAllLinks(page, pageSize, userID)
}

func DeleteLink(linkID, userID uint) error {
	result := database.DB.Where("id = ? AND user_id = ?", linkID, userID).Delete(&models.Link{})
	if result.RowsAffected == 0 {
		return errors.New("link not found or you don't have permission to delete it")
	}
	return result.Error
}

func UpdateLink(linkID, userID uint, originalURL string, customCode string, expiresIn *time.Duration) (*models.Link, error) {
	var link models.Link
	result := database.DB.Where("id = ? AND user_id = ?", linkID, userID).First(&link)
	if result.Error != nil {
		return nil, errors.New("link not found or you don't have permission to update it")
	}

	if customCode != "" && customCode != link.ShortCode {
		var existingLink models.Link
		result := database.DB.Where("short_code = ? AND id != ?", customCode, linkID).First(&existingLink)
		if result.Error == nil {
			return nil, errors.New("custom short code already exists")
		} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, result.Error
		}
		link.ShortCode = customCode
	}

	if originalURL != "" {
		link.OriginalURL = originalURL
	}

	if expiresIn != nil {
		expiresAt := time.Now().Add(*expiresIn)
		link.ExpiresAt = &expiresAt
	}

	result = database.DB.Save(&link)
	if result.Error != nil {
		return nil, result.Error
	}

	return &link, nil
}

func GetClickStats(linkID, userID uint) ([]models.ClickStat, error) {
	var link models.Link
	result := database.DB.Where("id = ? AND user_id = ?", linkID, userID).First(&link)
	if result.Error != nil {
		return nil, errors.New("link not found or you don't have permission to view it")
	}

	var clickStats []models.ClickStat
	result = database.DB.Where("link_id = ?", linkID).Order("clicked_at desc").Find(&clickStats)
	if result.Error != nil {
		return nil, result.Error
	}

	return clickStats, nil
}

func GetUserProfile(userID uint) (*models.User, error) {
	var user models.User
	result := database.DB.Where("id = ?", userID).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func GetLinksByTag(tag string, userID uint, page, pageSize int) ([]models.Link, int64, error) {
	var links []models.Link
	var total int64

	query := database.DB.Table("links").
		Joins("INNER JOIN link_tags ON links.id = link_tags.link_id").
		Joins("INNER JOIN tags ON link_tags.tag_id = tags.id").
		Where("tags.name = ? AND links.user_id = ?", tag, userID)

	query.Count(&total)

	result := query.Limit(pageSize).Offset((page - 1) * pageSize).
		Order("links.created_at desc").
		Find(&links)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	return links, total, nil
}

func generateShortCode() (string, error) {
	code := make([]byte, codeLength)
	charsetLength := big.NewInt(int64(len(charset)))

	for i := 0; i < codeLength; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", err
		}
		code[i] = charset[randomIndex.Int64()]
	}

	return string(code), nil
}
