package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"tg-bot-go/models"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleAdminCommand 管理员命令处理函数
func (h *Handler) handleAdminCommand(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	// 检查发送者是否是管理员
	var admin models.WhitelistUser
	if err := h.DB.Where("user_id = ? AND is_admin = ?", chatID, true).First(&admin).Error; err != nil {
		msg := tgbotapi.NewMessage(chatID, "您没有管理员权限。")
		h.Bot.Send(msg)
		return
	}

	// 解析命令
	parts := strings.Fields(text)
	command := parts[0]

	// 处理不需要参数的命令
	if command == "/checkuser" && len(parts) == 1 {
		var users []models.WhitelistUser
		if err := h.DB.Find(&users).Error; err != nil {
			msg := tgbotapi.NewMessage(chatID, "获取用户列表失败。")
			h.Bot.Send(msg)
			return
		}

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
		h.Bot.Send(msg)
		return
	}

	if len(parts) < 2 {
		msg := tgbotapi.NewMessage(chatID, "格式错误。正确格式：\n/adduser <用户ID> [天数]\n/deleteuser <用户ID>\n/extend <用户ID> <天数>\n/checkuser [用户ID]")
		h.Bot.Send(msg)
		return
	}

	userID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "用户ID格式错误。")
		h.Bot.Send(msg)
		return
	}

	switch command {
	case "/checkuser":
		user, err := models.GetUserExpiry(h.DB, userID)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("用户 %d 不存在。", userID))
			h.Bot.Send(msg)
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
		h.Bot.Send(msg)

	case "/adduser":
		days := 1
		if len(parts) > 2 {
			if d, err := strconv.Atoi(parts[2]); err == nil && d > 0 {
				days = d
			}
		}

		if err := models.AddUserToWhitelist(h.DB, userID, false); err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("添加用户失败：%v", err))
			h.Bot.Send(msg)
			return
		}
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("成功添加用户 %d 到白名单，有效期 %d 天。", userID, days))
		h.Bot.Send(msg)

	case "/deleteuser":
		if err := models.DeleteUserFromWhitelist(h.DB, userID); err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("删除用户失败：%v", err))
			h.Bot.Send(msg)
			return
		}
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("成功从白名单中删除用户 %d。", userID))
		h.Bot.Send(msg)

	case "/extend":
		if len(parts) != 3 {
			msg := tgbotapi.NewMessage(chatID, "格式错误。正确格式：/extend <用户ID> <天数>")
			h.Bot.Send(msg)
			return
		}

		days, err := strconv.Atoi(parts[2])
		if err != nil || days <= 0 {
			msg := tgbotapi.NewMessage(chatID, "天数格式错误，请输入大于0的整数。")
			h.Bot.Send(msg)
			return
		}

		duration := time.Duration(days) * 24 * time.Hour
		if err := models.ExtendUserExpiry(h.DB, userID, duration); err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("延长用户有效期失败：%v", err))
			h.Bot.Send(msg)
			return
		}
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("成功延长用户 %d 的有效期 %d 天。", userID, days))
		h.Bot.Send(msg)
	}
}

// handleExpiryCommand 处理有效期查询
func (h *Handler) handleExpiryCommand(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID

	user, err := models.GetUserExpiry(h.DB, chatID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "您还不是白名单用户。")
		h.Bot.Send(msg)
		return
	}

	var messageText string
	if user.IsAdmin {
		messageText = "您是管理员用户，永久有效。"
	} else {
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
	h.Bot.Send(msg)
}