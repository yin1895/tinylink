# Dockerfile (项目根目录)

# 1. 构建阶段
FROM golang:alpine AS builder
WORKDIR /app

ENV GOPROXY=https://goproxy.cn,direct
# 复制依赖文件并下载
COPY go.mod go.sum ./
RUN go mod download
# 复制源码
COPY . .

# 定义构建参数，默认构建 API 服务
ARG APP_NAME=tinylink-api
# 编译 (CGO_ENABLED=0 构建静态二进制文件)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/main ./cmd/${APP_NAME}

# 2. 运行阶段
FROM alpine:latest
WORKDIR /app
# 从构建阶段复制二进制文件
COPY --from=builder /app/main .
# 启动命令
CMD ["/app/main"]