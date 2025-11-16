package storage

import (
	"context"
	"database/sql"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
)

var (
	Db  *sql.DB
	Rdb *redis.Client
	Ctx = context.Background()
)

// 初始化 MySQL 连接
func InitMySQL() (err error) {
	dsn := "root:root_password@tcp(127.0.0.1:33061)/tinylink?charset=utf8mb4&parseTime=True"
	Db, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	// 检查连接
	err = Db.Ping()
	if err != nil {
		return err
	}
	// 创建表 (如果不存在)
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS urls (
		id BIGINT NOT NULL AUTO_INCREMENT,
		long_url VARCHAR(2048) NOT NULL,
		PRIMARY KEY (id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	_, err = Db.Exec(createTableSQL)
	return err
}

// 初始化 Redis 连接
func InitRedis() {
	Rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
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

// 根据 ID 从 MySQL 获取长链接
func GetLongURL(id int64) (string, error) {
	var longURL string
	row := Db.QueryRow("SELECT long_url FROM urls WHERE id = ?", id)
	if err := row.Scan(&longURL); err != nil {
		return "", err
	}
	return longURL, nil
}
