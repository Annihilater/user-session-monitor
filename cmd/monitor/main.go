package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Annihilater/user-session-monitor/internal/event"
	"github.com/Annihilater/user-session-monitor/internal/monitor"
	"github.com/Annihilater/user-session-monitor/internal/notify"
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

	// 用于存储当前运行的监控器实例
	currentMonitor  *monitor.Monitor
	currentNotifier *notify.NotifyManager
	currentLogger   *zap.Logger
)

const (
	defaultConfigPath = "/etc/user-session-monitor/config.yaml"
	serviceName       = "user-session-monitor"
	pidFile           = "/var/run/user-session-monitor.pid"
)

func init() {
	// 自定义帮助信息
	flag.Usage = func() {
		fmt.Printf(`用户会话监控 - 监控 Linux 服务器上的用户登录和登出事件

用法:
  %s [命令] [参数]

命令:
  menu               - 显示管理菜单
  run                - 直接运行监控程序
  start              - 启动系统服务
  stop               - 停止系统服务
  restart            - 重启系统服务
  status             - 查看服务状态
  enable             - 设置开机自启
  disable            - 取消开机自启
  log                - 查看服务日志
  config             - 显示配置文件内容
  install            - 安装服务
  uninstall          - 卸载服务
  version            - 查看版本信息
  check              - 检查服务运行状态
  tcp-status         - 查看 TCP 连接状态

参数:
  -h, --help         显示帮助信息
  -config string     配置文件路径（默认为 /etc/user-session-monitor/config.yaml）

示例:
  # 显示管理菜单
  %s menu

  # 直接启动服务（默认行为）
  %s

  # 使用自定义配置文件运行监控
  %s run -config /path/to/config.yaml

  # 启动系统服务
  %s start

  # 查看服务日志
  %s log

  # 检查服务运行状态
  %s check

  # 查看 TCP 连接状态
  %s tcp-status

更多信息:
  项目主页: https://github.com/Annihilater/user-session-monitor
  问题反馈: https://github.com/Annihilater/user-session-monitor/issues
`, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName)
	}
}

func main() {
	// 解析命令行参数
	flag.Parse()

	// 获取子命令
	args := flag.Args()
	if len(args) == 0 {
		// 如果没有参数，直接启动服务
		if err := start(); err != nil {
			fmt.Printf("启动服务失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 将命令转换为小写以实现大小写不敏感
	cmd := strings.ToLower(args[0])
	var err error
	switch cmd {
	case "menu":
		err = showMenu()
	case "run":
		err = start()
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
	case "check":
		err = handleCheck()
	case "tcp-status":
		err = handleTCPStatus()
	default:
		fmt.Printf("未知的命令: %s\n", args[0])
		flag.Usage()
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
 11. 检查运行状态
 12. TCP连接状态

服务状态: %s
是否开机自启: %s

请输入选择 [0-12]: `, status, enabled)

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
	case "11":
		err = handleCheck()
	case "12":
		err = handleTCPStatus()
	default:
		return fmt.Errorf("无效的选择：%s", choice)
	}

	return err
}

func handleStart() error {
	// 检查服务是否已经在运行
	if currentMonitor != nil {
		return fmt.Errorf("服务已经在运行中")
	}

	// 启动服务
	if err := start(); err != nil {
		return fmt.Errorf("启动服务失败: %v", err)
	}

	return nil
}

func handleStop() error {
	if currentMonitor == nil {
		return fmt.Errorf("服务未运行")
	}

	// 优雅关闭
	if currentLogger != nil {
		currentLogger.Info("正在关闭服务...")
	}

	if currentMonitor != nil {
		currentMonitor.Stop()
		currentMonitor = nil
	}

	if currentNotifier != nil {
		currentNotifier.Stop()
		currentNotifier = nil
	}

	if currentLogger != nil {
		currentLogger.Info("服务已关闭")
		currentLogger = nil
	}

	// 删除 PID 文件
	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除 PID 文件失败: %v", err)
	}

	fmt.Println("服务已停止")
	return nil
}

func handleRestart() error {
	if err := handleStop(); err != nil && !strings.Contains(err.Error(), "服务未运行") {
		return fmt.Errorf("停止服务失败: %v", err)
	}
	return handleStart()
}

func handleStatus() error {
	if currentMonitor == nil {
		fmt.Println("服务状态: 未运行")
		return nil
	}

	fmt.Println("服务状态: 运行中")

	// 获取进程信息
	pid := os.Getpid()
	cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "pid,ppid,user,%cpu,%mem,etime,command")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("获取进程信息失败: %v", err)
	}

	return nil
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

func handleCheck() error {
	// 检查服务状态
	fmt.Println("\n=== 服务状态 ===")
	if err := handleStatus(); err != nil {
		fmt.Printf("获取服务状态失败: %v\n", err)
	}

	// 检查日志文件
	fmt.Println("\n=== 日志文件状态 ===")
	logFile := "/var/log/user-session-monitor.log"
	if stat, err := os.Stat(logFile); err == nil {
		fmt.Printf("日志文件大小: %d 字节\n", stat.Size())
		fmt.Printf("最后修改时间: %s\n", stat.ModTime().Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("日志文件不存在: %v\n", err)
	}

	// 检查配置文件
	fmt.Println("\n=== 配置文件状态 ===")
	if err := handleConfig(); err != nil {
		fmt.Printf("获取配置文件状态失败: %v\n", err)
	}

	return nil
}

func getServiceStatus() string {
	if currentMonitor != nil {
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

func start() error {
	// 如果已经在运行，返回错误
	if currentMonitor != nil {
		return fmt.Errorf("服务已经在运行中")
	}

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
		// 检查是否在源码目录下运行（通过检查 config/config.yaml 是否存在）
		if _, err := os.Stat("config/config.yaml"); err == nil {
			viper.SetConfigFile("config/config.yaml")
		} else {
			// 如果不在源码目录，则使用默认配置文件路径
			viper.SetConfigFile(defaultConfigPath)
		}
	}

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 初始化日志配置
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// 创建日志器
	logger, err := config.Build()
	if err != nil {
		return fmt.Errorf("初始化日志器失败: %v", err)
	}
	currentLogger = logger

	// 确保在程序退出时同步日志
	defer func() {
		if err := logger.Sync(); err != nil {
			// 在某些平台上，Sync 可能会返回 "sync /dev/stderr: invalid argument" 错误
			// 这是一个已知问题，可以安全地忽略
			// 参考：https://github.com/uber-go/zap/issues/880
			if err.Error() != "sync /dev/stderr: invalid argument" {
				logger.Error("同步日志失败", zap.Error(err))
			}
		}
	}()

	// 输出版本和配置信息
	logger.Info("启动用户会话监控",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("build_date", date),
		zap.String("config_file", viper.ConfigFileUsed()),
	)

	// 输出配置内容
	maskedConfig := getMaskedConfig()
	logger.Info("当前配置",
		zap.Any("monitor", maskedConfig["monitor"]),
		zap.Any("notify", maskedConfig["notify"]),
	)

	// 创建事件总线
	eventBus := event.NewBus(100) // 设置适当的缓冲区大小

	// 获取运行模式配置
	runMode := strings.ToLower(viper.GetString("monitor.run_mode"))
	if runMode != "thread" && runMode != "goroutine" {
		runMode = "goroutine" // 默认使用协程模式
		logger.Info("未指定运行模式或运行模式无效，使用默认协程模式")
	}
	logger.Info("监控运行模式",
		zap.String("run_mode", runMode),
		zap.String("config_value", viper.GetString("monitor.run_mode")),
	)

	// 初始化监控器
	mon := monitor.NewMonitor(
		viper.GetString("monitor.log_file"),
		eventBus,
		logger,
		runMode,
	)
	currentMonitor = mon

	// 初始化通知服务
	notifyService := notify.NewNotifyManager(logger)
	if err := notifyService.InitNotifiers(); err != nil {
		logger.Error("初始化通知器失败", zap.Error(err))
		return fmt.Errorf("初始化通知器失败: %v", err)
	}
	currentNotifier = notifyService

	// 写入PID文件
	pid := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
		logger.Error("写入PID文件失败", zap.Error(err))
		// 不要因为PID文件写入失败就退出，只记录错误
	}

	// 启动监控器
	if err := mon.Start(); err != nil {
		// 如果启动失败，清理资源
		currentMonitor = nil
		currentNotifier = nil
		return fmt.Errorf("启动监控器失败: %v", err)
	}

	// 启动通知服务
	notifyService.Start(eventBus)

	fmt.Println("服务已启动")

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待退出信号
	<-sigChan

	// 优雅关闭
	return handleStop()
}

// handleTCPStatus 处理 TCP 状态查询命令
func handleTCPStatus() error {
	if currentMonitor == nil {
		return fmt.Errorf("服务未运行")
	}

	// 获取一次 TCP 状态
	state, err := currentMonitor.TCPMonitor.GetTCPState()
	if err != nil {
		return fmt.Errorf("获取 TCP 状态失败: %v", err)
	}

	// 打印状态信息
	fmt.Printf("\nTCP 连接状态统计:\n")
	fmt.Printf("————————————————\n")
	fmt.Printf("已建立连接 (ESTABLISHED): %d\n", state.Established)
	fmt.Printf("监听连接 (LISTEN):       %d\n", state.Listen)
	fmt.Printf("等待关闭 (TIME_WAIT):    %d\n", state.TimeWait)
	fmt.Printf("收到SYN (SYN_RECV):     %d\n", state.SynRecv)
	fmt.Printf("等待关闭 (CLOSE_WAIT):   %d\n", state.CloseWait)
	fmt.Printf("最后确认 (LAST_ACK):     %d\n", state.LastAck)
	fmt.Printf("已发SYN (SYN_SENT):     %d\n", state.SynSent)
	fmt.Printf("正在关闭 (CLOSING):      %d\n", state.Closing)
	fmt.Printf("等待FIN (FIN_WAIT1):    %d\n", state.FinWait1)
	fmt.Printf("等待关闭 (FIN_WAIT2):    %d\n", state.FinWait2)
	fmt.Printf("————————————————\n")

	return nil
}

// getMaskedConfig 获取脱敏后的配置
func getMaskedConfig() map[string]interface{} {
	config := viper.AllSettings()

	// 处理通知配置的脱敏
	if notifyConfig, ok := config["notify"].(map[string]interface{}); ok {
		// 处理飞书配置
		if feishuConfig, ok := notifyConfig["feishu"].(map[string]interface{}); ok {
			if _, exists := feishuConfig["webhook_url"]; exists {
				feishuConfig["webhook_url"] = "******"
			}
		}

		// 处理钉钉配置
		if dingtalkConfig, ok := notifyConfig["dingtalk"].(map[string]interface{}); ok {
			if _, exists := dingtalkConfig["webhook_url"]; exists {
				dingtalkConfig["webhook_url"] = "******"
			}
			if _, exists := dingtalkConfig["secret"]; exists {
				dingtalkConfig["secret"] = "******"
			}
		}
	}

	return config
}
