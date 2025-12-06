package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"tg-bot-go/models"
	"time"

	"github.com/BurntSushi/toml"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB     *gorm.DB
	Config Configuration
)

type Configuration struct {
	Database DatabaseConfig
	Telegram TelegramConfig
	OpenAI   OpenAIConfig
	Redis    RedisConfig
	Presets  PresetConfig
	Admin    AdminConfig
}

type DatabaseConfig struct {
	Host     string
	User     string
	Password string
	DBName   string
	Port     string
	SSLMode  string
}

type TelegramConfig struct {
	BotToken string
}

type OpenAIConfig struct {
    APIURL string
    APIKey string
    Model  string
    HTTPReferer string
    XTitle      string
}

type RedisConfig struct {
	Addr string
}

type PresetItem struct {
	Button  string
	Command string
	Content string
}

type PresetConfig struct {
	Items []PresetItem
}

type AdminConfig struct {
	AdminUserIDs []int64 `toml:"admin_user_ids"`
}

func InitConfig() {
	// 从环境变量读取配置
	adminIDs := strings.Split(getEnvOrDefault("ADMIN_USER_IDS", ""), ",")
	var adminUserIDs []int64
	for _, idStr := range adminIDs {
		if idStr = strings.TrimSpace(idStr); idStr != "" {
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				adminUserIDs = append(adminUserIDs, id)
			}
		}
	}

	// 从 TOML 文件读取预设配置
	var presets PresetConfig
	if _, err := toml.DecodeFile("config/presets.toml", &presets); err != nil {
		log.Printf("Warning: Could not load presets.toml: %v", err)
		// 使用默认预设
		presets = PresetConfig{
			Items: []PresetItem{},
		}
	}

    Config = Configuration{
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			User:     getEnvOrDefault("DB_USER", "root"),
			Password: getEnvOrDefault("DB_PASSWORD", ""),
			DBName:   getEnvOrDefault("DB_NAME", "tg_bot"),
			Port:     getEnvOrDefault("DB_PORT", "5432"),
			SSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
		},
		Telegram: TelegramConfig{
			BotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		},
        OpenAI: OpenAIConfig{
            APIURL: getEnvOrDefault("OPENROUTER_API_URL", getEnvOrDefault("OPENAI_API_URL", "https://openrouter.ai/api")),
            APIKey: getEnvOrDefault("OPENROUTER_API_KEY", os.Getenv("OPENAI_API_KEY")),
            Model:  getEnvOrDefault("OPENROUTER_MODEL", getEnvOrDefault("OPENAI_MODEL", "openai/gpt-4o")),
            HTTPReferer: os.Getenv("OPENROUTER_HTTP_REFERER"),
            XTitle:      os.Getenv("OPENROUTER_X_TITLE"),
        },
		Redis: RedisConfig{
			Addr: fmt.Sprintf("%s:%s",
				getEnvOrDefault("REDIS_HOST", "localhost"),
				getEnvOrDefault("REDIS_PORT", "6379"),
			),
		},
		Presets: presets,
		Admin: AdminConfig{
			AdminUserIDs: adminUserIDs,
		},
	}

	// 验证必要的配置
	validateConfig()
}

func validateConfig() {
	if Config.Database.User == "" {
		log.Fatal("DB_USER is not set")
	}
	if Config.Database.Password == "" {
		log.Fatal("DB_PASSWORD is not set")
	}
	if Config.Database.DBName == "" {
		log.Fatal("DB_NAME is not set")
	}
	if Config.Telegram.BotToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not set")
	}
	if len(Config.Admin.AdminUserIDs) == 0 {
		log.Fatal("ADMIN_USER_IDS not set")
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultVal int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func InitDB() {
	dbConfig := Config.Database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		dbConfig.Host, dbConfig.User, dbConfig.Password, dbConfig.DBName, dbConfig.Port, dbConfig.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("无法连接到数据库：", err)
	}
	DB = db
}

func InitAdminUser() {
	if len(Config.Admin.AdminUserIDs) == 0 {
		log.Fatal("ADMIN_USER_IDS not set")
	}

	// 遍历所有管理员ID
	for _, adminID := range Config.Admin.AdminUserIDs {
		// 检查管理员是否已存在
		var adminUser models.WhitelistUser
		result := DB.Where("user_id = ?", adminID).First(&adminUser)

		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// 如果管理员不存在，则添加
				log.Printf("Adding admin user: %d", adminID)
				if err := models.AddUserToWhitelist(DB, adminID, true); err != nil {
					log.Printf("Warning: Failed to add admin user %d: %v", adminID, err)
					continue
				}
				log.Printf("Successfully added admin user: %d", adminID)
			} else {
				// 其他数据库错误
				log.Printf("Database error while checking admin user %d: %v", adminID, result.Error)
				continue
			}
		} else if !adminUser.IsAdmin {
			// 如果用户存在但不是管理员，更新为管理员
			log.Printf("Updating user %d to admin", adminID)
			if err := DB.Model(&adminUser).Updates(map[string]interface{}{
				"is_admin":   true,
				"expired_at": time.Now().AddDate(100, 0, 0),
			}).Error; err != nil {
				log.Printf("Warning: Failed to update user %d to admin: %v", adminID, err)
				continue
			}
			log.Printf("Successfully updated user %d to admin", adminID)
		} else {
			log.Printf("Admin user %d already exists", adminID)
		}
	}
}
