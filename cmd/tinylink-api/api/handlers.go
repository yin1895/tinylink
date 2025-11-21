package api

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yin1895/tinylink/internal/storage"
	pb "github.com/yin1895/tinylink/pkg/proto"
)

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var IdGenClient pb.IdGeneratorClient

func toBase62(num int64) string {
	var result []byte
	for num > 0 {
		rem := num % 62
		result = append(result, alphabet[rem])
		num = num / 62
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}

func fromBase62(str string) int64 {
	var result int64
	for _, char := range str {
		result = result * 62
		if '0' <= char && char <= '9' {
			result += int64(char - '0')
		} else if 'a' <= char && char <= 'z' {
			result += int64(char - 'a' + 10)
		} else if 'A' <= char && char <= 'Z' {
			result += int64(char - 'A' + 36)
		}
	}
	return result
}

func RedirectHandler(c *gin.Context) {
	shortCode := c.Param("shortURL")

	// --- 新增代码开始 ---
	// 0. 布隆过滤器拦截 (缓存穿透保护)
	// 这一步是性能优化的核心！
	exists, err := storage.BF.Exists(shortCode)
	if err != nil {
		// 如果 Redis 挂了或网络错误，为了保险起见，我们通常选择“放行”
		// 让请求继续走后续流程，避免因为防御组件故障导致服务不可用
		log.Printf("Bloom filter error: %v", err)
	} else if !exists {
		// 核心逻辑：如果布隆过滤器说“一定不存在”，那就一定不存在
		// 直接返回 404，甚至不需要去查 Redis 缓存，更不需要查 MySQL
		// 极大地保护了后端存储
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found (intercepted by bloom filter)"})
		return
	}
	// --- 新增代码结束 ---

	// 1. 先从 Redis 缓存中查找
	longURL, err := storage.Rdb.Get(storage.Ctx, shortCode).Result()
	if err == nil {
		// 缓存命中
		c.Redirect(http.StatusFound, longURL)
		return
	}

	// 2. 缓存未命中，解码 shortCode, 从 MySQL 中查找
	id := fromBase62(shortCode)
	longURL, err = storage.GetLongURL(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	// 3. 在 MySQL 中找到了，将其存入 Redis 缓存（设置1小时过期）
	storage.Rdb.Set(storage.Ctx, shortCode, longURL, 0) // 0 表示永不过期，也可设置为 time.Hour

	// 4. 重定向
	c.Redirect(http.StatusFound, longURL)
}

func ShortenURLHandler(c *gin.Context) {
	var json struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
		return
	}

	// ---- 核心逻辑改造 ----
	// 1. (新) 通过 gRPC 调用 id-generator 服务获取一个全局唯一的 ID
	// 我们创建一个空的请求，因为proto文件定义了请求是Empty
	req := &pb.Empty{}
	// 使用客户端调用 GenerateId 方法。通常会传递带超时的context。
	res, err := IdGenClient.GenerateId(context.Background(), req)
	if err != nil {
		// 如果gRPC调用失败，说明内部服务出错了，返回500错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to communicate with ID service"})
		return
	}
	// 从响应中获取ID
	id := res.GetId()

	// 2. (新) 使用获取到的ID和原始长链接，调用新的storage函数存入数据库
	if err := storage.SaveURLWithID(id, json.URL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save URL with new ID"})
		return
	}

	// 3. 将这个ID转换为Base62的短码
	shortCode := toBase62(id)

	// 4. 将生成的短码加入布隆过滤器
	// 我们把短码作为 key 加入，因为用户查询时是用短码查的
	if err := storage.BF.Add(shortCode); err != nil {
		// 为了可用性，选择记录日志并继续（降级策略）
		log.Printf("Failed to add to bloom filter: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"short_url": "http://localhost:8080/" + shortCode,
	})
}
