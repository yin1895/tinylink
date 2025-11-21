// cmd/tinylink-api/api/handlers.go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/yin1895/tinylink/internal/storage"
	pb "github.com/yin1895/tinylink/pkg/proto"

	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
)

var IdGenClient pb.IdGeneratorClient

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// ClickEvent 定义发送到 Kafka 的数据结构
type ClickEvent struct {
	ShortURL  string `json:"short_url"`
	LongURL   string `json:"long_url"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Timestamp int64  `json:"timestamp"`
}

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

	req := &pb.Empty{}
	res, err := IdGenClient.GenerateId(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to communicate with ID service"})
		return
	}
	id := res.GetId()

	if err := storage.SaveURLWithID(id, json.URL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save URL"})
		return
	}

	shortCode := toBase62(id)
	storage.BF.Add(shortCode) // 加入布隆过滤器

	c.JSON(http.StatusOK, gin.H{
		"short_url": "http://localhost:8080/" + shortCode,
	})
}

func RedirectHandler(c *gin.Context) {
	shortCode := c.Param("shortURL")

	// 1. 布隆过滤器拦截
	exists, err := storage.BF.Exists(shortCode)
	if err == nil && !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found (intercepted)"})
		return
	}

	// 2. 查缓存
	longURL, err := storage.Rdb.Get(storage.Ctx, shortCode).Result()
	if err != nil {
		// 3. 查数据库
		id := fromBase62(shortCode)
		longURL, err = storage.GetLongURL(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
			return
		}
		storage.Rdb.Set(storage.Ctx, shortCode, longURL, 0)
	}

	// 4. (核心) 异步发送分析数据到 Kafka
	// 使用 go func 让它不阻塞主线程，保证跳转速度极快
	go func(sUrl, lUrl, ip, ua string) {
		event := ClickEvent{
			ShortURL:  sUrl,
			LongURL:   lUrl,
			IP:        ip,
			UserAgent: ua,
			Timestamp: time.Now().Unix(),
		}
		jsonBytes, _ := json.Marshal(event)

		// 写入 Kafka
		storage.KafkaWriter.WriteMessages(context.Background(),
			kafka.Message{
				Value: jsonBytes,
			},
		)
	}(shortCode, longURL, c.ClientIP(), c.GetHeader("User-Agent"))

	// 5. 跳转
	c.Redirect(http.StatusFound, longURL)
}
