package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// Log 全局日志实例
var Log *logrus.Logger

// CustomFormatter 自定义日志格式
type CustomFormatter struct{}

// Format 实现 logrus.Formatter 接口
func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// 获取文件名和行号
	var fileLine string
	if entry.HasCaller() {
		fileName := filepath.Base(entry.Caller.File)
		fileLine = fmt.Sprintf("%s:%d", fileName, entry.Caller.Line)
	}

	// 格式化日志级别
	level := strings.ToUpper(entry.Level.String())
	// 对齐级别长度，例如 INFO, WARN, ERRO
	if len(level) > 4 {
		level = level[:4]
	}

	// 格式化时间
	timeStr := entry.Time.Format("2006-01-02 15:04:05")

	// 组装日志信息: [TIME] [LEVEL] [FILE:LINE] MSG
	msg := fmt.Sprintf("[%s] [%s] [%s] %s\n", timeStr, level, fileLine, entry.Message)

	return []byte(msg), nil
}

// InitLogger 初始化日志
func InitLogger(levelStr string, filePath string) error {
	Log = logrus.New()

	// 开启 ReportCaller 以获取文件名和行号
	Log.SetReportCaller(true)

	// 使用自定义 Formatter
	Log.SetFormatter(&CustomFormatter{})

	// 设置日志级别
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		level = logrus.InfoLevel // 默认级别
	}
	Log.SetLevel(level)

	// 设置输出：同时输出到控制台和文件
	writers := []io.Writer{os.Stdout}
	if filePath != "" {
		// 确保日志目录存在
		logDir := filepath.Dir(filePath)
		if logDir != "." {
			if err := os.MkdirAll(logDir, 0o755); err != nil {
				return fmt.Errorf("failed to create log directory: %w", err)
			}
		}

		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			return err
		}
		writers = append(writers, file)
	}
	Log.SetOutput(io.MultiWriter(writers...))

	return nil
}
