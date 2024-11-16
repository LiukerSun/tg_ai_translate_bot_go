package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type WhitelistUser struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    int64     `gorm:"uniqueIndex"`
	IsAdmin   bool      `gorm:"default:false"`
	ExpiredAt time.Time `gorm:"not null"`
}

// 自动迁移
func MigrateWhitelist(db *gorm.DB) {
	db.AutoMigrate(&WhitelistUser{})
}

// 添加新用户到白名单
func AddUserToWhitelist(db *gorm.DB, userID int64, isAdmin bool) error {
	expiredAt := time.Now().Add(24 * time.Hour)
	if isAdmin {
		expiredAt = time.Now().AddDate(100, 0, 0)
	}

	return db.Create(&WhitelistUser{
		UserID:    userID,
		IsAdmin:   isAdmin,
		ExpiredAt: expiredAt,
	}).Error
}

// 添加删除用户函数
func DeleteUserFromWhitelist(db *gorm.DB, userID int64) error {
	// 不允许删除管理员
	result := db.Where("user_id = ? AND is_admin = ?", userID, false).Delete(&WhitelistUser{})
	if result.RowsAffected == 0 {
		return fmt.Errorf("用户不存在或是管理员")
	}
	return result.Error
}

// 检查用户是否有效
func IsUserValid(db *gorm.DB, userID int64) (bool, error) {
	var user WhitelistUser
	if err := db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	// 管理员永久有效
	if user.IsAdmin {
		return true, nil
	}

	// 检查是否过期
	return time.Now().Before(user.ExpiredAt), nil
}

// 更新用户有效期
func ExtendUserExpiry(db *gorm.DB, userID int64, duration time.Duration) error {
	var user WhitelistUser
	if err := db.Where("user_id = ? AND is_admin = ?", userID, false).First(&user).Error; err != nil {
		return fmt.Errorf("用户不存在")
	}

	// 如果已过期，从当前时间开始计算
	var newExpiry time.Time
	if time.Now().After(user.ExpiredAt) {
		newExpiry = time.Now().Add(duration)
	} else {
		// 如果未过期，在原有时间基础上追加
		newExpiry = user.ExpiredAt.Add(duration)
	}

	return db.Model(&user).Update("expired_at", newExpiry).Error
}

// 获取用户有效期信息
func GetUserExpiry(db *gorm.DB, userID int64) (*WhitelistUser, error) {
	var user WhitelistUser
	if err := db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
