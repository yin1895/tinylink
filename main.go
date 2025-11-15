package main

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// 线程安全的map进行数据库储存模拟
var (
	urlStore = make(map[string]string)
	mu       sync.Mutex
)

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var counter int64 = 10000 //高起点避免短链接易被猜解

// 数字ID -> 62进制字符串
func toBase62(num int64) string {
	var result []byte
	for num > 0 {
		rem := num % 62
		result = append(result, alphabet[rem])
		num = num / 62
	}
	//reserse string
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}

func main() {
	//默认Gin引擎
	router := gin.Default()

	//根路由器处理器
	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "Welcome to TinyLink",
		})
	})

	//短链接API
	router.POST("/shorten", func(ctx *gin.Context) {
		var json struct {
			URL string `json:"url" binding:"required"`
		}
		if err := ctx.ShouldBindJSON(&json); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
			return
		}

		mu.Lock()
		counter++
		shortCode := toBase62(counter)
		urlStore[shortCode] = json.URL
		mu.Unlock()

		ctx.JSON(http.StatusOK, gin.H{
			"short_url": "http://localhost:8080/" + shortCode,
		})
	})

	//短链接重定向API
	router.GET("/:shortURL", func(ctx *gin.Context) {
		shortURl := ctx.Param("shortURL")

		mu.Lock()
		longURL, exists := urlStore[shortURl]
		mu.Unlock()

		if exists {
			ctx.Redirect(http.StatusFound, longURL)
		} else {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		}
	})

	//启动HTTP服务，监听 8080端口
	router.Run(":8080")
}
