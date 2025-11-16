package storage

import (
	"context"
	"database/sql"
	"fmt"

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
		id BIGINT NOT NULL,
		long_url VARCHAR(2048) NOT NULL,
		PRIMARY KEY (id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	_, err = Db.Exec(createTableSQL)
	if err != nil {
		return err
	}

	// 为 ID 生成器创建一个专门的票据表
	createTicketTableSQL := `
    CREATE TABLE IF NOT EXISTS tickets (
        id BIGINT NOT NULL AUTO_INCREMENT,
        stub CHAR(1) NOT NULL DEFAULT 'a',
        PRIMARY KEY (id)
    ) ENGINE=InnoDB;` // 使用 InnoDB 引擎保证事务安全
	_, err = Db.Exec(createTicketTableSQL)
	if err != nil {
		// 如果创建失败，返回详细错误
		return fmt.Errorf("failed to create tickets table: %w", err)
	}
	return nil
}

// 用于从数据库获取下一个全局唯一ID
func GetNextID() (int64, error) {
	// 插入一条记录并获取其自增ID，这是一个原子操作，能保证并发安全
	res, err := Db.Exec("INSERT INTO tickets (stub) VALUES ('a')")
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
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

// 根据一个给定的ID来保存URL
func SaveURLWithID(id int64, longURL string) error {
	_, err := Db.Exec("INSERT INTO urls(id, long_url) VALUES(?, ?)", id, longURL)
	return err
}
