// internal/storage/storage.go
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os" // 引入 os 包读取环境变量

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/segmentio/kafka-go" // 引入 kafka 包
)

var (
	Db          *sql.DB
	Rdb         *redis.Client
	Ctx         = context.Background()
	BF          *BloomFilter
	KafkaWriter *kafka.Writer // 全局 Kafka 写入器
)

// InitMySQL 初始化 MySQL 连接 (支持环境变量)
func InitMySQL() (err error) {
	// 获取环境变量，默认值为 localhost:33061 (本地开发用)
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "127.0.0.1:33061"
	}

	dsn := fmt.Sprintf("root:root_password@tcp(%s)/tinylink?charset=utf8mb4&parseTime=True", dbHost)
	Db, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	err = Db.Ping()
	if err != nil {
		return err
	}

	// 创建 urls 表
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS urls (
		id BIGINT NOT NULL,
		long_url VARCHAR(2048) NOT NULL,
		PRIMARY KEY (id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
	if _, err = Db.Exec(createTableSQL); err != nil {
		return err
	}

	// 创建 tickets 表
	createTicketTableSQL := `
	CREATE TABLE IF NOT EXISTS tickets (
		id BIGINT NOT NULL AUTO_INCREMENT,
		stub CHAR(1) NOT NULL DEFAULT 'a',
		PRIMARY KEY (id)
	) ENGINE=InnoDB;`
	if _, err = Db.Exec(createTicketTableSQL); err != nil {
		return err
	}

	return nil
}

// InitRedis 初始化 Redis 连接 (支持环境变量)
func InitRedis() error {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost:6379"
	}

	Rdb = redis.NewClient(&redis.Options{
		Addr:     redisHost,
		Password: "",
		DB:       0,
	})

	if _, err := Rdb.Ping(Ctx).Result(); err != nil {
		return err
	}
	log.Println("Successfully connected to Redis at", redisHost)
	return nil
}

// InitKafka 初始化 Kafka Producer (新增)
func InitKafka() {
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" {
		kafkaBroker = "localhost:9092"
	}

	log.Printf("Connecting to Kafka at %s...", kafkaBroker)

	KafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    "link_clicks", // 消息发送到这个 Topic
		Balancer: &kafka.LeastBytes{},
		Async:    true, // 异步发送，不阻塞主流程
	}
}

// 将长链接存入 MySQL 并返回自增 ID
func SaveLongURL(longURL string) (int64, error) {
	res, err := Db.Exec("INSERT INTO urls(long_url) VALUES(?)", longURL)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetNextID 获取下一个全局唯一 ID
func GetNextID() (int64, error) {
	res, err := Db.Exec("INSERT INTO tickets (stub) VALUES ('a')")
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// SaveURLWithID 保存 ID 和 URL 的映射
func SaveURLWithID(id int64, longURL string) error {
	_, err := Db.Exec("INSERT INTO urls(id, long_url) VALUES(?, ?)", id, longURL)
	return err
}

// GetLongURL 根据 ID 获取长链接
func GetLongURL(id int64) (string, error) {
	var longURL string
	row := Db.QueryRow("SELECT long_url FROM urls WHERE id = ?", id)
	if err := row.Scan(&longURL); err != nil {
		return "", err
	}
	return longURL, nil
}
