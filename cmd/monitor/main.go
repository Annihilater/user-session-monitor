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
	if len(args) == 0 {
		showMenu()
		return
	}

	// 将命令转换为小写以实现大小写不敏感
	cmd := strings.ToLower(args[0])
	switch cmd {
	case "start":
		handleStart()
	case "stop":
		handleStop()
	case "restart":
		handleRestart()
	case "status":
		handleStatus()
	case "enable":
		handleEnable()
	case "disable":
		handleDisable()
	case "log":
		handleLog()
	case "config":
		handleConfig()
	case "install":
		handleInstall()
	case "uninstall":
		handleUninstall()
	case "version":
		handleVersion()
	default:
		fmt.Printf("未知的命令: %s\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func showMenu() {
	// 获取服务状态
	status := getServiceStatus()
	enabled := isServiceEnabled()

	fmt.Printf(`
  用户会话监控管理脚本
--- https://github.com/Annihilater/user-session-monitor ---
  0. 修改配置
————————————————
  1. 安装服务
  2. 卸载服务
————————————————
  3. 启动服务
  4. 停止服务
  5. 重启服务
  6. 查看服务状态
  7. 查看服务日志
————————————————
  8. 设置开机自启
  9. 取消开机自启
————————————————
 10. 查看版本信息

服务状态: %s
是否开机自启: %s

请输入选择 [0-10]: `, status, enabled)

	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "0":
		handleConfig()
	case "1":
		handleInstall()
	case "2":
		handleUninstall()
	case "3":
		handleStart()
	case "4":
		handleStop()
	case "5":
		handleRestart()
	case "6":
		handleStatus()
	case "7":
		handleLog()
	case "8":
		handleEnable()
	case "9":
		handleDisable()
	case "10":
		handleVersion()
	default:
		fmt.Println("无效的选择！")
	}
}

func printUsage() {
	fmt.Printf(`用户会话监控管理命令使用说明:
------------------------------------------
%s                    - 显示管理菜单 (功能更多)
%s start              - 启动服务
%s stop               - 停止服务
%s restart            - 重启服务
%s status             - 查看服务状态
%s enable             - 设置开机自启
%s disable            - 取消开机自启
%s log                - 查看服务日志
%s config             - 显示配置文件内容
%s install            - 安装服务
%s uninstall          - 卸载服务
%s version            - 查看版本信息
------------------------------------------
`, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName,
		serviceName, serviceName, serviceName, serviceName, serviceName, serviceName)
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

func handleStatus() {
	cmd := exec.Command("systemctl", "status", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func handleEnable() {
	cmd := exec.Command("systemctl", "enable", serviceName)
	if err := cmd.Run(); err != nil {
		fmt.Printf("设置开机自启失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("已设置开机自启")
}

func handleDisable() {
	cmd := exec.Command("systemctl", "disable", serviceName)
	if err := cmd.Run(); err != nil {
		fmt.Printf("取消开机自启失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("已取消开机自启")
}

func handleLog() {
	cmd := exec.Command("journalctl", "-u", serviceName, "-f")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func handleConfig() {
	configPath := *configFile
	if configPath == "" {
		configPath = defaultConfigPath
	}

	// 读取并显示配置文件内容
	content, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("读取配置文件失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("配置文件内容 (%s):\n%s\n", configPath, string(content))
}

func handleInstall() {
	fmt.Println("正在安装服务...")
	// 这里可以调用安装脚本或执行安装步骤
	fmt.Println("服务安装完成")
}

func handleUninstall() {
	fmt.Println("正在卸载服务...")
	// 这里可以调用卸载脚本或执行卸载步骤
	fmt.Println("服务卸载完成")
}

func handleVersion() {
	fmt.Printf("版本信息:\n")
	fmt.Printf("  版本号: %s\n", version)
	fmt.Printf("  构建时间: %s\n", date)
	fmt.Printf("  提交哈希: %s\n", commit)
}

func getServiceStatus() string {
	cmd := exec.Command("systemctl", "is-active", serviceName)
	output, _ := cmd.Output()
	status := strings.TrimSpace(string(output))
	if status == "active" {
		return "运行中"
	}
	return "未运行"
}

func isServiceEnabled() string {
	cmd := exec.Command("systemctl", "is-enabled", serviceName)
	output, _ := cmd.Output()
	enabled := strings.TrimSpace(string(output))
	if enabled == "enabled" {
		return "是"
	}
	return "否"
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
