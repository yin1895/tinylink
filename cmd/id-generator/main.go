// cmd/id-generator/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/yin1895/tinylink/internal/storage"
	// 引用我们生成的 proto 包，并给它起一个别名 pb
	pb "github.com/yin1895/tinylink/pkg/proto"

	"google.golang.org/grpc"
)

// 定义一个 server 结构体，我们将为它实现 gRPC 接口
type idGeneratorServer struct {
	pb.UnimplementedIdGeneratorServer
}

// 为 server 结构体实现我们在 .proto 文件中定义的 GenerateId 方法
func (s *idGeneratorServer) GenerateId(ctx context.Context, in *pb.Empty) (*pb.GenerateIdResponse, error) {
	log.Println("Received request to generate a new ID.")

	// 调用 storage 层的方法获取ID
	id, err := storage.GetNextID()
	if err != nil {
		log.Printf("Failed to get next ID from storage: %v", err)
		// 返回一个gRPC错误
		return nil, fmt.Errorf("internal storage error: %w", err)
	}

	log.Printf("Generated new ID: %d", id)
	// 将获取到的ID封装到响应消息中并返回
	return &pb.GenerateIdResponse{Id: id}, nil
}

func main() {
	// 1. 初始化数据库连接
	if err := storage.InitMySQL(); err != nil {
		log.Fatalf("Fatal: Failed to connect to MySQL: %v", err)
	}

	// 2. 监听一个TCP端口，gRPC服务将在这里等待连接
	// 50051 是 gRPC 的非官方标准端口
	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Fatal: Failed to listen on port %s: %v", port, err)
	}

	// 3. 创建一个新的 gRPC 服务器实例
	s := grpc.NewServer()

	// 4. 将我们的服务实现 (idGeneratorServer) 注册到 gRPC 服务器上
	pb.RegisterIdGeneratorServer(s, &idGeneratorServer{})

	log.Printf("gRPC server listening at %v", lis.Addr())

	// 5. 启动 gRPC 服务器，它会阻塞在这里，直到程序被终止
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Fatal: Failed to serve gRPC server: %v", err)
	}
}
