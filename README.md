# TinyLink - Distributed Short Link & Analytics Platform

TinyLink 是一个基于云原生架构的高性能分布式短链接平台。不仅实现了核心业务，更注重**高可用**、**高并发**与**可观测性**的工程实践。

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)
![Python](https://img.shields.io/badge/Python-3.9+-3776AB.svg)

## 🌟 核心亮点 (Key Features)

### 1. 架构设计
- **微服务拆分**: 采用 gRPC 拆分 **API 网关** 与 **ID 生成器**，实现职责分离。
- **异步解耦**: 引入 **Kafka** 构建事件驱动架构，将数据分析逻辑异步化，由 Python 消费者处理。

### 2. 性能与稳定性
- **多级缓存**: Redis 缓存热点数据，配合 **布隆过滤器 (Bloom Filter)** 彻底解决缓存穿透问题。
- **优雅停机**: 全链路实现 Graceful Shutdown，确保滚动更新时零请求丢失。

### 3. 可观测性 (Observability)
- **监控体系**: 集成 **Prometheus** 采集服务 RED 指标 (Request, Error, Duration)。
- **可视化**: 部署 **Grafana** 实时监控 QPS、响应延迟与系统健康度。

## 🛠 技术栈

- **Backend**：Golang (Gin, gRPC)，Python 3
- **Storage**：MySQL 8.0，Redis 6.2
- **Message Queue**：Kafka，Zookeeper
- **Infrastructure**：Docker，Docker Compose

## 🏗 架构图

<pre>
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
</pre>

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
