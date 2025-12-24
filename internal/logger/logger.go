package logger

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// Log 全局日志实例
var Log *logrus.Logger

// InitLogger 初始化日志
func InitLogger(levelStr string, filePath string) error {
	Log = logrus.New()

	// 设置日志格式
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 设置日志级别
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		level = logrus.InfoLevel // 默认级别
	}
	Log.SetLevel(level)

	// 设置输出：同时输出到控制台和文件
	writers := []io.Writer{os.Stdout}
	if filePath != "" {
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		writers = append(writers, file)
	}
	Log.SetOutput(io.MultiWriter(writers...))

	return nil
}
