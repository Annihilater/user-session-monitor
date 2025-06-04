package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	// 命令行参数
	configFile = flag.String("config", "", "配置文件路径，默认为 /etc/user-session-monitor/config.yaml")
)

const (
	defaultConfigPath = "/etc/user-session-monitor/config.yaml"
	serviceName       = "user-session-monitor"
)

func main() {
	// 解析命令行参数
	flag.Parse()

	// 获取子命令
	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "start":
			handleStart()
		case "stop":
			handleStop()
		case "restart":
			handleRestart()
		case "log":
			handleLog()
		case "info":
			handleInfo()
		default:
			fmt.Printf("未知的命令: %s\n", args[0])
			printUsage()
			os.Exit(1)
		}
		return
	}

	// 如果没有子命令，则启动监控程序
	startMonitor()
}

func printUsage() {
	fmt.Printf(`用法: %s <命令>

可用命令:
  start    启动服务
  stop     停止服务
  restart  重启服务
  log      查看服务日志
  info     查看服务状态和配置信息

选项:
  -config string   配置文件路径（默认为 /etc/user-session-monitor/config.yaml）

示例:
  %s start
  %s -config /custom/path/config.yaml start
`, serviceName, serviceName, serviceName)
}

func handleStart() {
	cmd := exec.Command("systemctl", "start", serviceName)
	if err := cmd.Run(); err != nil {
		fmt.Printf("启动服务失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("服务已启动")
}

func handleStop() {
	cmd := exec.Command("systemctl", "stop", serviceName)
	if err := cmd.Run(); err != nil {
		fmt.Printf("停止服务失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("服务已停止")
}

func handleRestart() {
	cmd := exec.Command("systemctl", "restart", serviceName)
	if err := cmd.Run(); err != nil {
		fmt.Printf("重启服务失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("服务已重启")
}

func handleLog() {
	cmd := exec.Command("journalctl", "-u", serviceName, "-f")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("查看日志失败: %v\n", err)
		os.Exit(1)
	}
}

func handleInfo() {
	// 检查服务状态
	status := exec.Command("systemctl", "is-active", serviceName)
	statusOutput, _ := status.Output()
	isActive := strings.TrimSpace(string(statusOutput)) == "active"

	// 获取进程 ID
	var pid string
	if isActive {
		pidCmd := exec.Command("systemctl", "show", "--property=MainPID", serviceName)
		pidOutput, _ := pidCmd.Output()
		pid = strings.TrimPrefix(strings.TrimSpace(string(pidOutput)), "MainPID=")
	}

	// 获取配置文件路径
	configPath := *configFile
	if configPath == "" {
		configPath = defaultConfigPath
	}

	// 输出信息
	fmt.Printf("服务信息:\n")
	fmt.Printf("  版本: %s\n", version)
	fmt.Printf("  构建时间: %s\n", date)
	fmt.Printf("  提交哈希: %s\n", commit)
	fmt.Printf("  状态: %s\n", strings.TrimSpace(string(statusOutput)))
	if pid != "" && pid != "0" {
		fmt.Printf("  进程ID: %s\n", pid)
	}
	fmt.Printf("  配置文件: %s\n", configPath)

	// 如果服务正在运行，尝试读取并显示配置内容
	if isActive {
		viper.Reset()
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err == nil {
			fmt.Printf("\n配置内容:\n")
			fmt.Printf("  日志文件: %s\n", viper.GetString("monitor.log_file"))
			webhookURL := viper.GetString("feishu.webhook_url")
			if webhookURL != "" {
				// 隐藏 webhook URL 的大部分内容
				maskedURL := webhookURL[:10] + "..." + webhookURL[len(webhookURL)-10:]
				fmt.Printf("  飞书 Webhook: %s\n", maskedURL)
			}
		}
	}
}

func startMonitor() {
	// 初始化配置
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// 如果指定了配置文件路径，则使用指定的路径
	if *configFile != "" {
		// 获取配置文件的绝对路径
		absPath, err := filepath.Abs(*configFile)
		if err != nil {
			panic("无法获取配置文件的绝对路径: " + err.Error())
		}
		// 设置配置文件路径
		viper.SetConfigFile(absPath)
	} else {
		// 使用默认配置文件路径
		viper.SetConfigFile(defaultConfigPath)
	}

	// 初始化日志配置
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// 创建日志器
	logger, err := config.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	// 确保在程序退出时同步日志
	defer func() {
		if err := logger.Sync(); err != nil {
			// 在某些平台上，Sync 可能会返回 "sync /dev/stderr: invalid argument" 错误
			// 这是一个已知问题，可以安全地忽略
			// 参考：https://github.com/uber-go/zap/issues/880
			if err.Error() != "sync /dev/stderr: invalid argument" {
				logger.Error("failed to sync logger", zap.Error(err))
			}
		}
	}()

	// 输出版本信息
	logger.Info("starting user session monitor",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("build_date", date),
		zap.String("config_file", viper.ConfigFileUsed()),
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
