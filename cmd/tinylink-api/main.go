// main.go
package main

import (
	"log"

	api "github.com/yin1895/tinylink/cmd/tinylink-api/api"
	"github.com/yin1895/tinylink/internal/storage"
	pb "github.com/yin1895/tinylink/pkg/proto"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure" //无TLS，本地开发
)

func main() {
	// 1. 初始化数据库连接
	if err := storage.InitMySQL(); err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	// 2. 初始化 Redis
	// 注意：如果之前加了 error 返回值，这里记得处理 err
	if err := storage.InitRedis(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	storage.BF = storage.NewBloomFilter("tinylink:bloom_filter", 1000000, 0.01)
	log.Println("Bloom Filter initialized.")
	//初始化gRPC
	idServiceAddr := "localhost:50051"

	conn, err := grpc.NewClient(idServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server at %s: %v", idServiceAddr, err)
	}

	//释放资源
	defer conn.Close()

	//创建IdGenerateClient
	idClient := pb.NewIdGeneratorClient(conn)
	// 客户端实例“注入”到 api 包的全局变量中
	api.IdGenClient = idClient

	log.Println("Successfully connected to gRPC ID Generator service.")

	// 2. 设置 Gin 路由
	router := gin.Default()
	router.POST("/shorten", api.ShortenURLHandler)
	router.GET("/:shortURL", api.RedirectHandler)

	// 3. 启动服务
	router.Run(":8080")
}
