package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"tg-bot-go/config"
	"tg-bot-go/logger"
	"tg-bot-go/models"
	"tg-bot-go/openai"
	"time"

	"github.com/go-redis/redis/v8"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var ctx = context.Background()
var rdb = redis.NewClient(&redis.Options{
	Addr: config.Config.Redis.Addr, // 使用配置文件中的 Redis 地址
})

// 添加常量定义
const (
	MAX_CONTEXT_LENGTH = 3000 // 最大上下文长度
)

func HandleMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	// 检查是否是命令（以/开头）
	if strings.HasPrefix(text, "/") {
		handleCommand(bot, update)
		return
	}
	// 检查用户是否在白名单中且未过期
	isValid, validErr := models.IsUserValid(config.DB, chatID)
	if validErr != nil {
		logger.LogRuntime(fmt.Sprintf("检查用户有效性失败：%v", validErr))
		msg := tgbotapi.NewMessage(chatID, "系统错误，请稍后再试。")
		bot.Send(msg)
		return
	}
	if !isValid {
		msg := tgbotapi.NewMessage(chatID, "您的使用权限已过期或未获得授权。")
		bot.Send(msg)
		return
	}

	// 记录用户消息
	logger.LogUserMessage(chatID, text)

	// 获取用户上下文
	contextKey := fmt.Sprintf("user:%d:context", chatID)
	presetKey := fmt.Sprintf("user:%d:preset", chatID)

	// 获取用户预设（作为 system prompt）
	userPreset, _ := rdb.Get(ctx, presetKey).Result()

	var response string
	var err error

	if userPreset == "" {
		// 如果没有预设，使用普通对话模式
		response, err = openai.GetOpenAIResponse("你是一个有帮助的助手。", text)
	} else {
		// 如果有预设，使用预设的翻译模式
		response, err = openai.GetOpenAIResponse(userPreset, text)
	}

	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "获取响应失败，请稍后再试。")
		bot.Send(msg)
		return
	}

	// 发送响应消息
	msg := tgbotapi.NewMessage(chatID, response)
	bot.Send(msg)

	// 记录回复内容
	logger.LogUserMessage(chatID, response)

	// 更新对话上下文
	var newContext string
	userContext, _ := rdb.Get(ctx, contextKey).Result()
	if userContext == "" {
		newContext = text + "\n" + response
	} else {
		newContext = userContext + "\n" + text + "\n" + response
	}

	// 检查上下文长度
	if len(newContext) > MAX_CONTEXT_LENGTH {
		// 清空对话上下文，但保留预设
		if err := rdb.Del(ctx, contextKey).Err(); err != nil {
			logger.LogRuntime(fmt.Sprintf("Failed to delete context: %v", err))
		}
		// 通知用户
		warningMsg := tgbotapi.NewMessage(chatID, "对话长度已超过限制，已自动清空对话历史。您可以继续对话，当前的预设模式不变。")
		bot.Send(warningMsg)
		// 保存当前对话作为新的上下文
		newContext = text + "\n" + response
	}

	// 保存新的上下文
	if err := rdb.Set(ctx, contextKey, newContext, 15*time.Minute).Err(); err != nil {
		logger.LogRuntime(fmt.Sprintf("Failed to update context: %v", err))
	}
}

// 新增命令处理函数
func handleCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
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
			bot.Send(inlineMsg)

			// 添加底部键盘
			clearKeyboard := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("/clear"),
				),
			)
			clearKeyboard.ResizeKeyboard = true // 设置键盘大小自适应
			return
		}

		bot.Send(msg)

	case "/clear":
		// 清空对话上下文和预设
		contextKey := fmt.Sprintf("user:%d:context", chatID)
		presetKey := fmt.Sprintf("user:%d:preset", chatID)

		if err := rdb.Del(ctx, contextKey, presetKey).Err(); err != nil {
			logger.LogRuntime(fmt.Sprintf("Failed to delete Redis context and preset: %v", err))
			msg := tgbotapi.NewMessage(chatID, "清空上下文失败，请稍后再试。")
			bot.Send(msg)
			return
		}

		msg := tgbotapi.NewMessage(chatID, "已清空所有对话上下文和预设。")
		// 添加 inline keyboard
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
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
		bot.Send(msg)

	case "/expiry":
		handleExpiryCommand(bot, update)

	case "/id":
		// 获取用户ID
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("您的用户ID是：%d", chatID))
		bot.Send(msg)

	case "/adduser", "/deleteuser", "/extend", "/checkuser":
		handleAdminCommand(bot, update)

	default:
		// 处理预设命令
		for _, item := range config.Config.Presets.Items {
			if command == item.Command {
				handlePresetCommand(bot, update, item)
				return
			}
		}
	}
}

// 新增预设命令处理函数
func handlePresetCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update, preset config.PresetItem) {
	chatID := update.Message.Chat.ID

	// 保存用户选择的预设
	presetKey := fmt.Sprintf("user:%d:preset", chatID)
	if err := rdb.Set(ctx, presetKey, preset.Content, 24*time.Hour).Err(); err != nil {
		logger.LogRuntime(fmt.Sprintf("Failed to save preset: %v", err))
		msg := tgbotapi.NewMessage(chatID, "设置预设失败，请稍后再试。")
		bot.Send(msg)
		return
	}

	// 清空对话上下文
	contextKey := fmt.Sprintf("user:%d:context", chatID)
	if err := rdb.Del(ctx, contextKey).Err(); err != nil {
		logger.LogRuntime(fmt.Sprintf("Failed to delete context: %v", err))
	}

	// 只发送简单的确认消息，不需要 inline keyboard
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("已切换到%s，您可以开始对话了。", preset.Button))
	bot.Send(msg)
}

// 新增管理员命令处理函数
func handleAdminCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	// 检查发送者是否是管理员
	var admin models.WhitelistUser
	if err := config.DB.Where("user_id = ? AND is_admin = ?", chatID, true).First(&admin).Error; err != nil {
		msg := tgbotapi.NewMessage(chatID, "您没有管理员权限。")
		bot.Send(msg)
		return
	}

	// 解析命令
	parts := strings.Fields(text)
	command := parts[0]

	// 处理不需要参数的命令
	if command == "/checkuser" && len(parts) == 1 {
		// 获取所有用户列表
		var users []models.WhitelistUser
		if err := config.DB.Find(&users).Error; err != nil {
			msg := tgbotapi.NewMessage(chatID, "获取用户列表失败。")
			bot.Send(msg)
			return
		}

		// 构建用户列表消息
		var messageText strings.Builder
		messageText.WriteString("当前用户列表：\n\n")
		for _, user := range users {
			if user.IsAdmin {
				messageText.WriteString(fmt.Sprintf("用户 ID: %d (管理员)\n", user.UserID))
			} else {
				remainingTime := user.ExpiredAt.Sub(time.Now())
				if remainingTime <= 0 {
					messageText.WriteString(fmt.Sprintf("用户 ID: %d (已过期)\n", user.UserID))
				} else {
					days := int(remainingTime.Hours() / 24)
					hours := int(remainingTime.Hours()) % 24
					messageText.WriteString(fmt.Sprintf("用户 ID: %d (剩余 %d 天 %d 小时)\n", user.UserID, days, hours))
				}
			}
		}
		messageText.WriteString("\n使用 /checkuser <用户ID> 查看指定用户详细信息")

		msg := tgbotapi.NewMessage(chatID, messageText.String())
		bot.Send(msg)
		return
	}

	// 处理需要参数的命令
	if len(parts) < 2 {
		msg := tgbotapi.NewMessage(chatID, "格式错误。正确格式：\n/adduser <用户ID> [天数]\n/deleteuser <用户ID>\n/extend <用户ID> <天数>\n/checkuser [用户ID]")
		bot.Send(msg)
		return
	}

	// 解析用户ID
	userID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "用户ID格式错误。")
		bot.Send(msg)
		return
	}

	switch command {
	case "/checkuser":
		// 获取指定用户信息
		user, err := models.GetUserExpiry(config.DB, userID)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("用户 %d 不存在。", userID))
			bot.Send(msg)
			return
		}

		var messageText string
		if user.IsAdmin {
			messageText = fmt.Sprintf("用户 %d 是管理员用户，永久有效。", userID)
		} else {
			remainingTime := user.ExpiredAt.Sub(time.Now())
			if remainingTime <= 0 {
				messageText = fmt.Sprintf("用户 %d 的使用权限已过期。\n过期时间：%v", userID, user.ExpiredAt.Format("2006-01-02 15:04:05"))
			} else {
				days := int(remainingTime.Hours() / 24)
				hours := int(remainingTime.Hours()) % 24
				messageText = fmt.Sprintf("用户 %d 的使用权限还剩 %d 天 %d 小时。\n到期时间：%v",
					userID, days, hours, user.ExpiredAt.Format("2006-01-02 15:04:05"))
			}
		}
		msg := tgbotapi.NewMessage(chatID, messageText)
		bot.Send(msg)

	case "/adduser":
		// 解析天数（可选参数）
		days := 1 // 默认1天
		if len(parts) > 2 {
			if d, err := strconv.Atoi(parts[2]); err == nil && d > 0 {
				days = d
			}
		}

		// 添加用户到白名单
		if err := models.AddUserToWhitelist(config.DB, userID, false); err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("添加用户失败：%v", err))
			bot.Send(msg)
			return
		}
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("成功添加用户 %d 到白名单，有效期 %d 天。", userID, days))
		bot.Send(msg)

	case "/deleteuser":
		// 删除用户
		if err := models.DeleteUserFromWhitelist(config.DB, userID); err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("删除用户失败：%v", err))
			bot.Send(msg)
			return
		}
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("成功从白名单中删除用户 %d。", userID))
		bot.Send(msg)

	case "/extend":
		// 检查是否提供了天数
		if len(parts) != 3 {
			msg := tgbotapi.NewMessage(chatID, "格式错误。正确格式：/extend <用户ID> <天数>")
			bot.Send(msg)
			return
		}

		days, err := strconv.Atoi(parts[2])
		if err != nil || days <= 0 {
			msg := tgbotapi.NewMessage(chatID, "天数格式错误，请输入大于0的整数。")
			bot.Send(msg)
			return
		}

		// 延长用户有效期
		duration := time.Duration(days) * 24 * time.Hour
		if err := models.ExtendUserExpiry(config.DB, userID, duration); err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("延长用户有效期失败：%v", err))
			bot.Send(msg)
			return
		}
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("成功延长用户 %d 的有效期 %d 天。", userID, days))
		bot.Send(msg)
	}
}

// 添加回调处理函数
func HandleCallback(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	callback := update.CallbackQuery
	chatID := callback.Message.Chat.ID
	data := callback.Data

	// 检查用户是否在白名单中
	var whitelistUser models.WhitelistUser
	if err := config.DB.Where("user_id = ?", chatID).First(&whitelistUser).Error; err != nil {
		msg := tgbotapi.NewMessage(chatID, "您没有权限使用此机器人。")
		bot.Send(msg)
		return
	}

	switch data {
	case "/help":
		msg := tgbotapi.NewMessage(chatID, "这里是帮助信息...")
		bot.Send(msg)
	case "/about":
		msg := tgbotapi.NewMessage(chatID, "关于我们...")
		bot.Send(msg)
	default:
		// 处理预设命令
		for _, item := range config.Config.Presets.Items {
			if data == item.Command {
				// 保存用户选择的预设
				presetKey := fmt.Sprintf("user:%d:preset", chatID)
				if err := rdb.Set(ctx, presetKey, item.Content, 24*time.Hour).Err(); err != nil {
					logger.LogRuntime(fmt.Sprintf("Failed to save preset: %v", err))
					msg := tgbotapi.NewMessage(chatID, "设置预设失败，请稍后再试。")
					bot.Send(msg)
					return
				}

				// 清空对话上下文
				contextKey := fmt.Sprintf("user:%d:context", chatID)
				if err := rdb.Del(ctx, contextKey).Err(); err != nil {
					logger.LogRuntime(fmt.Sprintf("Failed to delete context: %v", err))
				}

				// 只发送简单的确认消息，不需要 inline keyboard
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("已切换到%s，您可以开始对话了。", item.Button))
				bot.Send(msg)
				return
			}
		}
	}

	// 回应回调查询
	callbackResponse := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
	if _, err := bot.Request(callbackResponse); err != nil {
		logger.LogRuntime(fmt.Sprintf("Error answering callback query: %v", err))
	}
}

// 添加处理有效期查询的函数
func handleExpiryCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	chatID := update.Message.Chat.ID

	// 获取用户信息
	user, err := models.GetUserExpiry(config.DB, chatID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "您还不是白名单用户。")
		bot.Send(msg)
		return
	}

	// 格式化消息
	var messageText string
	if user.IsAdmin {
		messageText = "您是管理员用户，永久有效。"
	} else {
		// 计算剩余时间
		remainingTime := user.ExpiredAt.Sub(time.Now())
		if remainingTime <= 0 {
			messageText = "您的使用权限已过期。"
		} else {
			days := int(remainingTime.Hours() / 24)
			hours := int(remainingTime.Hours()) % 24
			messageText = fmt.Sprintf("您的使用权限还剩 %d 天 %d 小时。", days, hours)
		}
	}

	msg := tgbotapi.NewMessage(chatID, messageText)
	bot.Send(msg)
}
