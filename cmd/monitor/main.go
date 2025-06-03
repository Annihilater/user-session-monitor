package main

import (
	"log"
	"os"

	"github.com/spf13/viper"

	"github.com/Annihilater/user-session-monitor/internal/feishu"
	"github.com/Annihilater/user-session-monitor/internal/monitor"
)

func main() {
	// 初始化配置
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	// 初始化日志
	logger := log.Default()

	// 初始化飞书通知器
	notifier := feishu.NewNotifier(viper.GetString("feishu.webhook_url"))

	// 初始化监控器
	mon := monitor.NewMonitor(
		viper.GetString("monitor.log_file"),
		notifier,
		logger,
	)

	// 启动监控
	log.Println("Starting user session monitor...")
	if err := mon.Start(); err != nil {
		log.Println("Monitor failed", err)
		os.Exit(1)
	}
}
