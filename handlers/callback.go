package handlers

import (
	"fmt"
	"tg-bot-go/config"
	"tg-bot-go/logger"
	"tg-bot-go/models"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleCallback 处理回调查询
func (h *Handler) HandleCallback(update tgbotapi.Update) {
	callback := update.CallbackQuery
	chatID := callback.Message.Chat.ID
	data := callback.Data

	// 检查用户是否在白名单中
	var whitelistUser models.WhitelistUser
	if err := h.DB.Where("user_id = ?", chatID).First(&whitelistUser).Error; err != nil {
		msg := tgbotapi.NewMessage(chatID, "您没有权限使用此机器人。")
		h.Bot.Send(msg)
		return
	}

	switch data {
	case "/help":
		msg := tgbotapi.NewMessage(chatID, "这里是帮助信息...")
		h.Bot.Send(msg)
	case "/about":
		msg := tgbotapi.NewMessage(chatID, "关于我们...")
		h.Bot.Send(msg)
	default:
		// 处理预设命令
		for _, item := range config.Config.Presets.Items {
			if data == item.Command {
				// 保存用户选择的预设
				presetKey := fmt.Sprintf("user:%d:preset", chatID)
				if err := h.Redis.Set(ctx, presetKey, item.Content, 24*time.Hour).Err(); err != nil {
					logger.LogRuntime(fmt.Sprintf("Failed to save preset: %v", err))
					msg := tgbotapi.NewMessage(chatID, "设置预设失败，请稍后再试。")
					h.Bot.Send(msg)
					return
				}

				// 清空对话上下文
				contextKey := fmt.Sprintf("user:%d:context", chatID)
				if err := h.Redis.Del(ctx, contextKey).Err(); err != nil {
					logger.LogRuntime(fmt.Sprintf("Failed to delete context: %v", err))
				}

				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("已切换到%s，您可以开始对话了。", item.Button))
				h.Bot.Send(msg)
				return
			}
		}
	}

	// 回应回调查询
	callbackResponse := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
	if _, err := h.Bot.Request(callbackResponse); err != nil {
		logger.LogRuntime(fmt.Sprintf("Error answering callback query: %v", err))
	}
}