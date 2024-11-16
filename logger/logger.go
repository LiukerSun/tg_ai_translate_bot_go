package logger

import (
	"log"
	"os"
	"path/filepath"
)

var (
	combinedLog *log.Logger
)

func init() {
	// 创建日志目录
	if err := os.MkdirAll("logs", os.ModePerm); err != nil {
		log.Fatalf("无法创建日志目录：%v", err)
	}

	// 初始化综合日志
	combinedLogFile, err := os.OpenFile(filepath.Join("logs", "combined.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("无法创建综合日志文件：%v", err)
	}
	combinedLog = log.New(combinedLogFile, "", log.Ldate|log.Ltime|log.Lshortfile)
}

func LogRuntime(message string) {
	combinedLog.Printf("RUNTIME: %s", message)
}

func LogAPI(message string) {
	combinedLog.Printf("API: %s", message)
}

func LogUserMessage(userID int64, message string) {
	combinedLog.Printf("USER %d: %s", userID, message)
}
