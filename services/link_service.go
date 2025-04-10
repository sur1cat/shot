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

func CreateShortLink(originalURL string, customCode string, expiresIn *time.Duration) (*models.Link, error) {
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

func GetAllLinks(page, pageSize int) ([]models.Link, int64, error) {
	var links []models.Link
	var total int64

	database.DB.Model(&models.Link{}).Count(&total)

	result := database.DB.Limit(pageSize).Offset((page - 1) * pageSize).Order("created_at desc").Find(&links)
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
