package api

import (
	"net/http"

	"github.com/yin1895/tinylink/storage"

	"github.com/gin-gonic/gin"
)

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func toBase62(num int64) string {
	var result []byte
	for num > 0 {
		rem := num % 62
		result = append(result, alphabet[rem])
		num = num / 62
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}

func fromBase62(str string) int64 {
	var result int64
	for _, char := range str {
		result = result * 62
		if '0' <= char && char <= '9' {
			result += int64(char - '0')
		} else if 'a' <= char && char <= 'z' {
			result += int64(char - 'a' + 10)
		} else if 'A' <= char && char <= 'Z' {
			result += int64(char - 'A' + 36)
		}
	}
	return result
}

func ShortenURLHandler(c *gin.Context) {
	var json struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
		return
	}

	// 1. 将长链接存入 MySQL 并获取 ID
	id, err := storage.SaveLongURL(json.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save URL"})
		return
	}

	// 2. 将 ID 转换为 Base62 的短码
	shortCode := toBase62(id)

	c.JSON(http.StatusOK, gin.H{
		"short_url": "http://localhost:8080/" + shortCode,
	})
}

func RedirectHandler(c *gin.Context) {
	shortCode := c.Param("shortURL")

	// 1. 先从 Redis 缓存中查找
	longURL, err := storage.Rdb.Get(storage.Ctx, shortCode).Result()
	if err == nil {
		// 缓存命中
		c.Redirect(http.StatusFound, longURL)
		return
	}

	// 2. 缓存未命中，解码 shortCode, 从 MySQL 中查找
	id := fromBase62(shortCode)
	longURL, err = storage.GetLongURL(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	// 3. 在 MySQL 中找到了，将其存入 Redis 缓存（设置1小时过期）
	storage.Rdb.Set(storage.Ctx, shortCode, longURL, 0) // 0 表示永不过期，也可设置为 time.Hour

	// 4. 重定向
	c.Redirect(http.StatusFound, longURL)
}
