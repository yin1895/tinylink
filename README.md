# TinyLink - Distributed Short Link & Analytics Platform

TinyLink 是一个基于微服务架构的高性能短链接生成与数据分析平台。

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)
![Python](https://img.shields.io/badge/Python-3.9+-3776AB.svg)

## 🚀 核心特性

- **微服务架构**：核心 API (Go/Gin) 与 发号器 (Go/gRPC) 分离。
- **高性能存储**：MySQL 持久化 + Redis 缓存 + 布隆过滤器（防止缓存穿透）。
- **异步数据分析**：基于 Kafka 的事件驱动架构，使用 Python 消费者进行流量分析（IP/User-Agent）。
- **工业级稳定性**：全链路实现优雅停机 (Graceful Shutdown)。
- **容器化部署**：基于 Docker Compose 的一键部署环境。

## 🛠 技术栈

- **Backend**：Golang (Gin, gRPC)，Python 3
- **Storage**：MySQL 8.0，Redis 6.2
- **Message Queue**：Kafka，Zookeeper
- **Infrastructure**：Docker，Docker Compose

## 🏗 架构图

User Request -> [Nginx/API Gateway]  
                      |  
      -----------------------------------  
      |               |                 |  
[Redis Cache] <-> [Go API] -----> [Kafka Producer]  
      ^               |                 |  
      |               v                 v  
[Bloom Filter]    [gRPC ID Gen]    [Kafka Cluster]  
                      |                 |  
                   [MySQL]              v  
                                  [Python Consumer]  
                                        |  
                                  [MySQL Analytics]

## ⚡️ 快速开始

### 前置要求
- Docker & Docker Compose

### 一键运行
```bash
# 克隆项目
git clone https://github.com/YOUR_USERNAME/tinylink.git
cd tinylink

# 启动所有服务
docker-compose up --build -d
```

### 测试接口

1. 生成短链接（POST）
```bash
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"url":"https://github.com"}'
```

2. 访问短链接  
在浏览器中访问：http://localhost:8080/{short_code}

## 📂 目录结构

```text
├── analytics/          # Python 数据分析微服务
├── cmd/
│   ├── tinylink-api/   # HTTP API 网关入口
│   └── id-generator/   # gRPC ID 生成器入口
├── internal/
│   └── storage/        # 数据库、缓存、布隆过滤器逻辑
├── pkg/
│   └── proto/          # gRPC Protobuf 定义
└── docker-compose.yml  # 容器编排文件
```
