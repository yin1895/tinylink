// main.go
package main

import (
	"log"

	"github.com/yin1895/tinylink/api"     // 引用 api 包
	"github.com/yin1895/tinylink/storage" // 引用 storage 包

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 初始化数据库连接
	if err := storage.InitMySQL(); err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	storage.InitRedis()

	// 2. 设置 Gin 路由
	router := gin.Default()
	router.POST("/shorten", api.ShortenURLHandler)
	router.GET("/:shortURL", api.RedirectHandler)

	// 3. 启动服务
	router.Run(":8080")
}
