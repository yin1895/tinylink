package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	//默认Gin引擎
	router := gin.Default()

	//根路由器处理器
	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "Welcome to TinyLink",
		})
	})

	//启动HTTP服务，监听 8080端口
	router.Run(":8080")
}
