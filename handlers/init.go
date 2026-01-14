package handlers

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

var (
	ctx = context.Background()
)

// Handler 结构体用于依赖注入
type Handler struct {
	Bot   *tgbotapi.BotAPI
	DB    *gorm.DB
	Redis *redis.Client
}

// NewHandler 创建新的处理程序实例
func NewHandler(bot *tgbotapi.BotAPI, db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{
		Bot:   bot,
		DB:    db,
		Redis: rdb,
	}
}

// InitRedis 初始化 Redis 客户端
func InitRedis(redisAddr string) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// 测试连接
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}
	return rdb
}

const (
	MAX_CONTEXT_LENGTH = 3000 // 最大上下文长度 (chars)
)