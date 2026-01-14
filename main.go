// main.go
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"tg-bot-go/config"
	"tg-bot-go/handlers"
	"tg-bot-go/logger"
	"tg-bot-go/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var ctx = context.Background()

func main() {
	// 添加开发模式标志
	devMode := flag.Bool("dev", false, "Run in development mode")
	flag.Parse()

	// 根据运行模式加载环境变量
	if *devMode {
		log.Println("Running in development mode")
		// 开发模式使用本地地址和本地环境变量
		if err := loadDevEnv(); err != nil {
			log.Printf("Warning: Could not load .env.dev file: %v", err)
		}
	}

	// 初始化配置
	config.InitConfig()

	// 初始化数据库
	config.InitDB()
	models.MigrateWhitelist(config.DB)

	// 初始化管理员
	config.InitAdminUser()

	// 初始化 Redis 客户端
	rdb := handlers.InitRedis(config.Config.Redis.Addr)

	// 清理 Redis 缓存
	if err := rdb.FlushDB(ctx).Err(); err != nil {
		log.Fatalf("无法清理 Redis 缓存：%v", err)
	}

	// 获取 Telegram Bot Token
	botToken := config.Config.Telegram.BotToken
	if botToken == "" {
		log.Fatal("Telegram Bot Token 未设置")
	}

	// 创建 Telegram Bot 实例
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	// 初始化 Handler (依赖注入)
	h := handlers.NewHandler(bot, config.DB, rdb)

	// 删除 Webhook
	_, err = bot.Request(tgbotapi.DeleteWebhookConfig{})
	if err != nil {
		log.Fatalf("删除 Webhook 失败：%v", err)
	}

	// 设置长轮询
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// 记录启动日志
	logger.LogRuntime("Bot started successfully")

	// 创建一个 WaitGroup 来管理 goroutines
	var wg sync.WaitGroup
	// 创建一个用户消息处理的通道，限制并发数
	maxConcurrent := 10 // 最大并发处理数
	sem := make(chan struct{}, maxConcurrent)

	// 处理消息
	for update := range updates {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量

		go func(update tgbotapi.Update) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			// 使用 recover 来防止 goroutine 崩溃
			defer func() {
				if r := recover(); r != nil {
					logger.LogRuntime(fmt.Sprintf("Recovered from panic in message handler: %v", r))
				}
			}()

			if update.Message != nil {
				h.HandleMessage(update)
			} else if update.CallbackQuery != nil {
				h.HandleCallback(update)
			}
		}(update)
	}

	// 等待所有 goroutines 完成
	wg.Wait()
}

// 加载开发环境配置
func loadDevEnv() error {
	log.Println("Loading .env.dev file...")
	file, err := os.Open(".env.dev")
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// 移除可能的引号
		value = strings.Trim(value, `"'`)
		os.Setenv(key, value)
		log.Printf("Set env: %s=%s", key, value)
	}

	return scanner.Err()
}
