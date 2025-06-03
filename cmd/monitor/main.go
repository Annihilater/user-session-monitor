package main

import (
	"os"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Annihilater/user-session-monitor/internal/feishu"
	"github.com/Annihilater/user-session-monitor/internal/monitor"
)

var (
	// 这些变量会在编译时通过 -ldflags 注入
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// 初始化配置
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")

	// 初始化日志配置
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// 创建日志器
	logger, err := config.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	// 输出版本信息
	logger.Info("starting user session monitor",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("build_date", date),
	)

	if err := viper.ReadInConfig(); err != nil {
		logger.Fatal("failed to read config",
			zap.Error(err),
		)
	}

	// 初始化飞书通知器
	notifier := feishu.NewNotifier(viper.GetString("feishu.webhook_url"))

	// 初始化监控器
	mon := monitor.NewMonitor(
		viper.GetString("monitor.log_file"),
		notifier,
		logger,
	)

	// 启动监控
	logger.Info("starting user session monitor")
	if err := mon.Start(); err != nil {
		logger.Error("monitor failed",
			zap.Error(err),
		)
		os.Exit(1)
	}
}
