package handlers

import (
	"fmt"
	"strings"
	"tg-bot-go/config"
	"tg-bot-go/logger"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleCommand 处理通用命令
func (h *Handler) handleCommand(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	// 获取命令部分（第一个空格前的内容）
	command := strings.Fields(text)[0]

	switch command {
	case "/start", "/help", "/about":
		var responseText string
		switch command {
		case "/start":
			responseText = "欢迎使用tg-bot-go！请选择一个选项："
		case "/help":
			responseText = "这里是帮助信息..."
		case "/about":
			responseText = "关于我们..."
		}

		msg := tgbotapi.NewMessage(chatID, responseText)

		if command == "/start" {
			// 创建 Inline Keyboard
			var buttons [][]tgbotapi.InlineKeyboardButton
			for _, item := range config.Config.Presets.Items {
				button := tgbotapi.NewInlineKeyboardButtonData(item.Button, item.Command)
				buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
			}

			// 添加帮助和关于按钮
			helpAboutRow := []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("帮助", "/help"),
				tgbotapi.NewInlineKeyboardButtonData("关于", "/about"),
			}
			buttons = append(buttons, helpAboutRow)

			inlineMsg := tgbotapi.NewMessage(chatID, responseText)
			inlineMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
			h.Bot.Send(inlineMsg)

			// 添加底部键盘
			clearKeyboard := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("/clear"),
				),
			)
			clearKeyboard.ResizeKeyboard = true
			return
		}

		h.Bot.Send(msg)

	case "/clear":
		// 清空对话上下文和预设
		contextKey := fmt.Sprintf("user:%d:context", chatID)
		presetKey := fmt.Sprintf("user:%d:preset", chatID)

		if err := h.Redis.Del(ctx, contextKey, presetKey).Err(); err != nil {
			logger.LogRuntime(fmt.Sprintf("Failed to delete Redis context and preset: %v", err))
			msg := tgbotapi.NewMessage(chatID, "清空上下文失败，请稍后再试。")
			h.Bot.Send(msg)
			return
		}

		msg := tgbotapi.NewMessage(chatID, "已清空所有对话上下文和预设。")
		var buttons [][]tgbotapi.InlineKeyboardButton
		for _, item := range config.Config.Presets.Items {
			button := tgbotapi.NewInlineKeyboardButtonData(item.Button, item.Command)
			buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
		}
		helpAboutRow := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("帮助", "/help"),
			tgbotapi.NewInlineKeyboardButtonData("关于", "/about"),
		}
		buttons = append(buttons, helpAboutRow)
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
		h.Bot.Send(msg)

	case "/expiry":
		h.handleExpiryCommand(update)

	case "/id":
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("您的用户ID是：%d", chatID))
		h.Bot.Send(msg)

	case "/adduser", "/deleteuser", "/extend", "/checkuser":
		h.handleAdminCommand(update)

	default:
		// 处理预设命令
		for _, item := range config.Config.Presets.Items {
			if command == item.Command {
				h.handlePresetCommand(update, item)
				return
			}
		}
	}
}

// handlePresetCommand 处理预设命令
func (h *Handler) handlePresetCommand(update tgbotapi.Update, preset config.PresetItem) {
	chatID := update.Message.Chat.ID

	// 保存用户选择的预设
	presetKey := fmt.Sprintf("user:%d:preset", chatID)
	if err := h.Redis.Set(ctx, presetKey, preset.Content, 24*time.Hour).Err(); err != nil {
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

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("已切换到%s，您可以开始对话了。", preset.Button))
	h.Bot.Send(msg)
}