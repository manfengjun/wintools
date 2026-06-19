package common

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const logDir = "logs"

type LogLevel int

const (
	LevelInfo  LogLevel = 0
	LevelWarn  LogLevel = 1
	LevelError LogLevel = 2
)

func (l LogLevel) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "DEBUG"
	}
}

var logFile *os.File

// InitLogger 初始化日志文件，写入 %APPDATA%\DesktopSuite\logs\app.log
func InitLogger() {
	dir := filepath.Join(ConfigDir(), logDir)
	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "app.log")

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// 无法写日志时 fallback 到 stderr
		fmt.Fprintf(os.Stderr, "[logger] failed to open log file: %v\n", err)
		return
	}
	logFile = f
}

// CloseLogger 关闭日志文件
func CloseLogger() {
	if logFile != nil {
		logFile.Close()
	}
}

func writeLog(level LogLevel, msg string, args ...interface{}) {
	if logFile == nil {
		return
	}
	ts := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("%s [%s] %s\n", ts, level, fmt.Sprintf(msg, args...))
	logFile.WriteString(line)
}

// Info 写入 INFO 级别日志
func Info(msg string, args ...interface{}) {
	writeLog(LevelInfo, msg, args...)
}

// Warn 写入 WARN 级别日志
func Warn(msg string, args ...interface{}) {
	writeLog(LevelWarn, msg, args...)
}

// Error 写入 ERROR 级别日志
func Error(msg string, args ...interface{}) {
	writeLog(LevelError, msg, args...)
}
