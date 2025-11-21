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

	"github.com/prometheus/client_golang/prometheus/promhttp"
	api "github.com/yin1895/tinylink/cmd/tinylink-api/api"
	"github.com/yin1895/tinylink/cmd/tinylink-api/middleware"
	"github.com/yin1895/tinylink/internal/storage"
	pb "github.com/yin1895/tinylink/pkg/proto"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 1. 初始化 MySQL
	if err := storage.InitMySQL(); err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer func() {
		if storage.Db != nil {
			storage.Db.Close()
		}
	}()

	// 2. 初始化 Redis
	if err := storage.InitRedis(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer func() {
		if storage.Rdb != nil {
			storage.Rdb.Close()
		}
	}()

	// 3. 初始化 Kafka (新增)
	storage.InitKafka()
	defer func() {
		if storage.KafkaWriter != nil {
			storage.KafkaWriter.Close()
		}
	}()

	// 4. 初始化布隆过滤器
	storage.BF = storage.NewBloomFilter("tinylink:bloom_filter", 1000000, 0.01)

	// 5. 连接 ID 生成器服务 (支持环境变量)
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

	// 6. 启动 HTTP 服务
	router := gin.Default()

	// (新) 注册监控中间件
	router.Use(middleware.PrometheusMiddleware())

	router.POST("/shorten", api.ShortenURLHandler)
	router.GET("/:shortURL", api.RedirectHandler)

	// (新) 暴露 Prometheus 指标接口
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		log.Println("Starting HTTP server on :8080 ...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 7. 优雅停机
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}
	log.Println("Server exiting")
}
