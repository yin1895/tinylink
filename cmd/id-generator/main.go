// cmd/id-generator/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/yin1895/tinylink/internal/storage"
	pb "github.com/yin1895/tinylink/pkg/proto"

	"google.golang.org/grpc"
)

type idGeneratorServer struct {
	pb.UnimplementedIdGeneratorServer
}

func (s *idGeneratorServer) GenerateId(ctx context.Context, in *pb.Empty) (*pb.GenerateIdResponse, error) {
	log.Println("Received request to generate a new ID.")
	id, err := storage.GetNextID()
	if err != nil {
		log.Printf("Failed to get next ID: %v", err)
		return nil, fmt.Errorf("internal storage error: %w", err)
	}
	return &pb.GenerateIdResponse{Id: id}, nil
}

func main() {
	// 1. 初始化数据库
	if err := storage.InitMySQL(); err != nil {
		log.Fatalf("Fatal: Failed to connect to MySQL: %v", err)
	}
	// 退出时关闭数据库连接
	defer func() {
		if storage.Db != nil {
			storage.Db.Close()
			log.Println("MySQL connection closed.")
		}
	}()

	// 2. 监听端口
	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Fatal: Failed to listen on port %s: %v", port, err)
	}

	// 3. 创建 gRPC 服务器
	s := grpc.NewServer()
	pb.RegisterIdGeneratorServer(s, &idGeneratorServer{})

	// 4. 在 Goroutine 中启动服务
	go func() {
		log.Printf("gRPC server listening at %v", lis.Addr())
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 5. 监听停止信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down gRPC server...")

	// 6. 停机逻辑
	// GracefulStop 会停止接收新连接，并阻塞直到所有待处理的 RPC 调用完成
	s.GracefulStop()

	log.Println("gRPC server stopped.")
}
