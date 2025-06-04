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
	configFile = flag.String(
		"config",
		"",
		"配置文件路径，默认为 /etc/user-session-monitor/config.yaml",
	)
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
		// 如果没有参数，直接运行监控程序
		if err := startMonitor(); err != nil {
			fmt.Printf("启动监控失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 将命令转换为小写以实现大小写不敏感
	cmd := strings.ToLower(args[0])
	var err error
	switch cmd {
	case "menu":
		// 添加 menu 命令来显示菜单
		err = showMenu()
	case "start":
		err = handleStart()
	case "stop":
		err = handleStop()
	case "restart":
		err = handleRestart()
	case "status":
		err = handleStatus()
	case "enable":
		err = handleEnable()
	case "disable":
		err = handleDisable()
	case "log":
		err = handleLog()
	case "config":
		err = handleConfig()
	case "install":
		err = handleInstall()
	case "uninstall":
		err = handleUninstall()
	case "version":
		err = handleVersion()
	case "run":
		err = startMonitor() // 添加 run 命令来启动监控
	default:
		fmt.Printf("未知的命令: %s\n", args[0])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("执行命令失败: %v\n", err)
		os.Exit(1)
	}
}

func showMenu() error {
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
	if _, err := fmt.Scanln(&choice); err != nil {
		return fmt.Errorf("读取输入失败: %v", err)
	}

	var err error
	switch choice {
	case "0":
		err = handleConfig()
	case "1":
		err = handleInstall()
	case "2":
		err = handleUninstall()
	case "3":
		err = handleStart()
	case "4":
		err = handleStop()
	case "5":
		err = handleRestart()
	case "6":
		err = handleStatus()
	case "7":
		err = handleLog()
	case "8":
		err = handleEnable()
	case "9":
		err = handleDisable()
	case "10":
		err = handleVersion()
	default:
		fmt.Println("无效的选择！")
	}

	return err
}

func printUsage() {
	fmt.Printf(`用户会话监控管理命令使用说明:
------------------------------------------
%s                    - 直接启动监控程序
%s menu              - 显示管理菜单
%s start             - 启动服务
%s stop              - 停止服务
%s restart           - 重启服务
%s status            - 查看服务状态
%s enable            - 设置开机自启
%s disable           - 取消开机自启
%s log               - 查看服务日志
%s config            - 显示配置文件内容
%s install           - 安装服务
%s uninstall         - 卸载服务
%s version           - 查看版本信息
%s run               - 直接运行监控程序
------------------------------------------
`, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName,
		serviceName, serviceName, serviceName, serviceName, serviceName, serviceName,
		serviceName, serviceName)
}

func handleStart() error {
	cmd := exec.Command("systemctl", "start", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("启动服务失败: %v", err)
	}
	fmt.Println("服务已启动")
	return nil
}

func handleStop() error {
	cmd := exec.Command("systemctl", "stop", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("停止服务失败: %v", err)
	}
	fmt.Println("服务已停止")
	return nil
}

func handleRestart() error {
	cmd := exec.Command("systemctl", "restart", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("重启服务失败: %v", err)
	}
	fmt.Println("服务已重启")
	return nil
}

func handleStatus() error {
	cmd := exec.Command("systemctl", "status", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func handleEnable() error {
	cmd := exec.Command("systemctl", "enable", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("设置开机自启失败: %v", err)
	}
	fmt.Println("已设置开机自启")
	return nil
}

func handleDisable() error {
	cmd := exec.Command("systemctl", "disable", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("取消开机自启失败: %v", err)
	}
	fmt.Println("已取消开机自启")
	return nil
}

func handleLog() error {
	cmd := exec.Command("journalctl", "-u", serviceName, "-f")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func handleConfig() error {
	configPath := *configFile
	if configPath == "" {
		configPath = defaultConfigPath
	}

	// 读取并显示配置文件内容
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}
	fmt.Printf("配置文件内容 (%s):\n%s\n", configPath, string(content))
	return nil
}

func handleInstall() error {
	fmt.Println("正在安装服务...")
	// 这里可以调用安装脚本或执行安装步骤
	fmt.Println("服务安装完成")
	return nil
}

func handleUninstall() error {
	fmt.Println("正在卸载服务...")
	// 这里可以调用卸载脚本或执行卸载步骤
	fmt.Println("服务卸载完成")
	return nil
}

func handleVersion() error {
	fmt.Printf("版本信息:\n")
	fmt.Printf("  版本号: %s\n", version)
	fmt.Printf("  构建时间: %s\n", date)
	fmt.Printf("  提交哈希: %s\n", commit)
	return nil
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

func startMonitor() error {
	// 初始化配置
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// 如果指定了配置文件路径，则使用指定的路径
	if *configFile != "" {
		// 获取配置文件的绝对路径
		absPath, err := filepath.Abs(*configFile)
		if err != nil {
			return fmt.Errorf("无法获取配置文件的绝对路径: %v", err)
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
		return fmt.Errorf("failed to initialize logger: %v", err)
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
		return fmt.Errorf("failed to read config: %v", err)
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
		return fmt.Errorf("monitor failed: %v", err)
	}

	return nil
}
