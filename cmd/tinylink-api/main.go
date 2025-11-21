// cmd/tinylink-api/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	api "github.com/yin1895/tinylink/cmd/tinylink-api/api"
	"github.com/yin1895/tinylink/internal/storage"
	pb "github.com/yin1895/tinylink/pkg/proto"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 1. 初始化资源
	if err := storage.InitMySQL(); err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	// 记得程序退出前关闭数据库连接
	defer func() {
		if storage.Db != nil {
			storage.Db.Close()
			log.Println("MySQL connection closed.")
		}
	}()

	if err := storage.InitRedis(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	// 关闭 Redis 连接
	defer func() {
		if storage.Rdb != nil {
			storage.Rdb.Close()
			log.Println("Redis connection closed.")
		}
	}()

	// 初始化布隆过滤器
	storage.BF = storage.NewBloomFilter("tinylink:bloom_filter", 1000000, 0.01)
	log.Println("Bloom Filter initialized.")

	// 2. 连接 ID 生成器服务
	idServiceAddr := os.Getenv("ID_SERVICE_ADDR")
	if idServiceAddr == "" {
		idServiceAddr = "localhost:50051"
	}
	conn, err := grpc.Dial(idServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()
	api.IdGenClient = pb.NewIdGeneratorClient(conn)

	// 3. 配置 HTTP 服务器
	router := gin.Default()
	router.POST("/shorten", api.ShortenURLHandler)
	router.GET("/:shortURL", api.RedirectHandler)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// 4. 独立的 Goroutine 中启动服务器
	go func() {
		log.Println("Starting HTTP server on :8080 ...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 5. 监听系统信号 (优雅停机的核心)
	// 创建一个通道接收信号
	quit := make(chan os.Signal, 1)
	// 监听 SIGINT (Ctrl+C) 和 SIGTERM (Docker stop)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 程序阻塞直到收到信号
	<-quit
	log.Println("Shutting down server...")

	//6.停机逻辑
	// 创建一个 5 秒的超时上下文。

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}
