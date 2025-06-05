package monitor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/event"
	"github.com/Annihilater/user-session-monitor/internal/types"
)

// 系统认证日志文件路径
var authLogPaths = map[string]string{
	"debian":        "/var/log/auth.log", // Debian/Ubuntu
	"ubuntu":        "/var/log/auth.log", // Debian/Ubuntu
	"rhel":          "/var/log/secure",   // RHEL/CentOS
	"centos":        "/var/log/secure",   // RHEL/CentOS
	"fedora":        "/var/log/secure",   // Fedora
	"amazon":        "/var/log/secure",   // Amazon Linux
	"suse":          "/var/log/messages", // SUSE
	"opensuse-leap": "/var/log/messages", // openSUSE Leap
}

// 检测操作系统类型
func detectOSType() (string, error) {
	// 首先尝试读取 /etc/os-release 文件
	content, err := os.ReadFile("/etc/os-release")
	if err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "ID=") {
				osType := strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
				return strings.ToLower(osType), nil
			}
		}
	}

	// 如果无法从 os-release 获取，尝试其他发行版特定文件
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		return "debian", nil
	}
	if _, err := os.Stat("/etc/redhat-release"); err == nil {
		return "rhel", nil
	}
	if _, err := os.Stat("/etc/centos-release"); err == nil {
		return "centos", nil
	}

	return "", fmt.Errorf("无法检测操作系统类型")
}

// 获取认证日志文件路径
func getAuthLogPath(configPath string) (string, error) {
	// 如果配置文件中指定了路径，优先使用配置的路径
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	// 自动检测操作系统类型
	osType, err := detectOSType()
	if err != nil {
		return "", fmt.Errorf("检测操作系统类型失败: %v", err)
	}

	// 根据操作系统类型获取日志路径
	if logPath, ok := authLogPaths[osType]; ok {
		// 验证日志文件是否存在且可读
		if _, err := os.Stat(logPath); err == nil {
			return logPath, nil
		}
	}

	return "", fmt.Errorf("无法找到认证日志文件")
}

var (
	// 登录事件匹配模式
	// 匹配示例：
	// sshd[0000000]: Accepted publickey for root from 192.168.1.1 port 55030 ssh2: RSA SHA256:xxxxxxxxxxx
	// 匹配组说明：
	// (\w+) - 第一个组：用户名
	// ([\d\.]+) - 第二个组：IP地址
	// (\d+) - 第三个组：端口号
	// 支持的认证方式：password（密码认证）和 publickey（密钥认证）
	loginPattern = regexp.MustCompile(`(?m)sshd\[\d+\]: Accepted (?:password|publickey) for (\w+) from ([\d\.]+) port (\d+)`)

	// 登出事件匹配模式列表
	// 由于登出事件有多种不同的日志格式，这里使用多个正则表达式进行匹配
	logoutPatterns = []*regexp.Regexp{
		// 1. 用户主动断开连接场景
		// 匹配示例：sshd[0000000]: Received disconnect from 192.168.1.1 port 55030:11: disconnected by user
		// 匹配组说明：
		// ([\d\.]+) - 第一个组：IP地址
		// (\d+) - 第二个组：端口号
		// 常见于以下情况：
		// - 用户执行 exit 命令
		// - 用户执行 logout 命令
		// - 用户按 Ctrl + D
		// - SSH 客户端正常关闭
		regexp.MustCompile(`(?m)sshd\[\d+\]: Received disconnect from ([\d\.]+) port (\d+):11: disconnected by user`),

		// 2. 用户断开连接场景（带用户名）
		// 匹配示例：sshd[0000000]: Disconnected from user root 192.168.1.1 port 55030
		// 匹配组说明：
		// (\w+) - 第一个组：用户名
		// ([\d\.]+) - 第二个组：IP地址
		// (\d+) - 第三个组：端口号
		// 常见于以下情况：
		// - SSH 会话正常结束
		// - 客户端网络断开
		// - 服务器端会话超时
		regexp.MustCompile(`(?m)sshd\[\d+\]: Disconnected from user (\w+) ([\d\.]+) port (\d+)`),

		// 3. PAM 会话关闭场景
		// 匹配示例：sshd[0000000]: pam_unix(sshd:session): session closed for user root
		// 匹配组说明：
		// (\w+) - 第一个组：用户名
		// 常见于以下情况：
		// - 系统强制关闭会话
		// - PAM 会话超时
		// - 系统关机或重启
		// 注意：此场景下无法直接获取 IP 和端口信息，需要从之前的登录记录中查找
		regexp.MustCompile(`(?m)sshd\[\d+\]: pam_unix\(sshd:session\): session closed for user (\w+)`),
	}

	// 用于存储最近的登录记录，用于补充登出信息
	// key 格式：username:ip:port
	// value: loginRecord 结构体，包含完整的会话信息
	// 主要用途：
	// 1. 用于关联登录和登出事件
	// 2. 补充某些登出场景下缺失的 IP 和端口信息
	// 3. 跟踪用户会话状态
	loginRecords = make(map[string]types.LoginRecord)

	// 用于存储最近的登出记录，用于去重
	// key 格式：username:ip:port
	// value: 最后一次登出时间
	logoutRecords     = make(map[string]time.Time)
	logoutRecordMutex sync.RWMutex

	// 登出事件的去重时间窗口
	logoutDeduplicationWindow = 5 * time.Second
)

// makeLoginKey 生成登录记录的唯一键
// 参数：
//   - username: 用户名
//   - ip: 登录源 IP
//   - port: 登录源端口
//
// 返回值：
//   - string: 格式为 "username:ip:port" 的唯一键
func makeLoginKey(username, ip, port string) string {
	return fmt.Sprintf("%s:%s:%s", username, ip, port)
}

// Monitor 监控器
type Monitor struct {
	logFile          string
	eventBus         *event.Bus
	logger           *zap.Logger
	stopChan         chan struct{}
	runMode          string            // 运行模式：thread 或 goroutine
	TCPMonitor       *TCPMonitor       // TCP 连接监控
	SystemMonitor    *SystemMonitor    // 系统资源监控
	HardwareMonitor  *HardwareMonitor  // 硬件信息监控
	HeartbeatMonitor *HeartbeatMonitor // 心跳监控
	NetworkMonitor   *NetworkMonitor   // 网络监控
	ProcessMonitor   *ProcessMonitor   // 进程监控
	ServerMonitor    *ServerMonitor    // 服务器信息监控
}

func NewMonitor(logFile string, eventBus *event.Bus, logger *zap.Logger, runMode string) *Monitor {
	// 默认使用协程模式
	if runMode != "thread" && runMode != "goroutine" {
		runMode = "goroutine"
	}
	return &Monitor{
		logFile:  logFile,
		eventBus: eventBus,
		logger:   logger,
		stopChan: make(chan struct{}),
		runMode:  runMode,
	}
}

func (m *Monitor) Start() error {
	// 获取认证日志文件路径
	logPath, err := getAuthLogPath(m.logFile)
	if err != nil {
		return fmt.Errorf("获取认证日志文件路径失败: %v", err)
	}
	m.logFile = logPath

	// 检查日志文件是否存在且可读
	if _, err := os.Stat(m.logFile); os.IsNotExist(err) {
		return fmt.Errorf("日志文件 %s 不存在", m.logFile)
	}

	// 尝试打开文件以验证权限
	file, err := os.Open(m.logFile)
	if err != nil {
		return fmt.Errorf("无法打开日志文件 %s: %v", m.logFile, err)
	}
	if err := file.Close(); err != nil {
		m.logger.Error("关闭日志文件失败",
			zap.String("file", m.logFile),
			zap.Error(err),
		)
	}

	// 获取服务器监控配置
	serverIntervalFloat := viper.GetFloat64("monitor.server.interval")
	serverInterval := time.Duration(serverIntervalFloat * float64(time.Second))
	if serverInterval < 100*time.Millisecond {
		serverInterval = time.Second // 默认1秒，最小100毫秒
		m.logger.Warn("服务器监控间隔太小，使用默认值", zap.Duration("interval", serverInterval))
	}

	// 启动服务器信息监控
	m.ServerMonitor = NewServerMonitor(m.logger, serverInterval, m.runMode)
	m.ServerMonitor.Start()

	// 获取初始服务器信息用于日志记录
	serverInfo, err := m.ServerMonitor.getServerInfo()
	if err != nil {
		return fmt.Errorf("获取服务器信息失败: %v", err)
	}
	m.logger.Info("服务器信息",
		zap.String("hostname", serverInfo.Hostname),
		zap.String("ip", serverInfo.IP),
		zap.String("os_type", serverInfo.OSType),
		zap.String("log_file", m.logFile),
	)

	// 获取监控配置
	tcpIntervalFloat := viper.GetFloat64("monitor.tcp.interval")
	sysIntervalFloat := viper.GetFloat64("monitor.system.interval")
	hwIntervalFloat := viper.GetFloat64("monitor.hardware.interval")
	heartbeatIntervalFloat := viper.GetFloat64("monitor.heartbeat.interval")

	// 记录读取到的配置
	m.logger.Info("读取监控配置",
		zap.Float64("tcp_interval_seconds", tcpIntervalFloat),
		zap.Float64("system_interval_seconds", sysIntervalFloat),
		zap.Float64("hardware_interval_seconds", hwIntervalFloat),
		zap.Float64("heartbeat_interval_seconds", heartbeatIntervalFloat),
	)

	// 转换为 Duration
	tcpInterval := time.Duration(tcpIntervalFloat * float64(time.Second))
	if tcpInterval < 100*time.Millisecond {
		tcpInterval = time.Second // 默认1秒，最小100毫秒
		m.logger.Warn("TCP监控间隔太小，使用默认值", zap.Duration("interval", tcpInterval))
	}

	sysInterval := time.Duration(sysIntervalFloat * float64(time.Second))
	if sysInterval < 100*time.Millisecond {
		sysInterval = 5 * time.Second // 默认5秒，最小100毫秒
		m.logger.Warn("系统监控间隔太小，使用默认值", zap.Duration("interval", sysInterval))
	}

	hwInterval := time.Duration(hwIntervalFloat * float64(time.Second))
	if hwInterval < 100*time.Millisecond {
		hwInterval = time.Second // 默认1秒，最小100毫秒
		m.logger.Warn("硬件监控间隔太小，使用默认值", zap.Duration("interval", hwInterval))
	}

	diskPaths := viper.GetStringSlice("monitor.system.disk_paths")
	if len(diskPaths) == 0 {
		diskPaths = []string{"/"} // 默认监控根目录
	}

	hwDiskPaths := viper.GetStringSlice("monitor.hardware.disk_paths")
	if len(hwDiskPaths) == 0 {
		hwDiskPaths = diskPaths // 默认使用系统监控的磁盘路径
	}

	// 处理心跳监控间隔
	heartbeatInterval := time.Duration(heartbeatIntervalFloat * float64(time.Second))
	if heartbeatInterval < 100*time.Millisecond {
		heartbeatInterval = time.Second // 默认1秒，最小100毫秒
		m.logger.Warn("心跳监控间隔太小，使用默认值", zap.Duration("interval", heartbeatInterval))
	}

	// 记录最终使用的配置
	m.logger.Info("使用监控配置",
		zap.Duration("tcp_interval", tcpInterval),
		zap.Duration("system_interval", sysInterval),
		zap.Duration("hardware_interval", hwInterval),
		zap.Duration("heartbeat_interval", heartbeatInterval),
		zap.Strings("disk_paths", diskPaths),
		zap.Strings("hardware_disk_paths", hwDiskPaths),
	)

	// 启动 TCP 监控
	m.TCPMonitor = NewTCPMonitor(m.logger, tcpInterval, m.runMode)
	m.TCPMonitor.Start()

	// 启动心跳监控
	m.HeartbeatMonitor = NewHeartbeatMonitor(m.logger, heartbeatInterval, m.runMode)
	m.HeartbeatMonitor.Start()

	// 获取网络监控配置
	networkIntervalFloat := viper.GetFloat64("monitor.network.interval")
	networkInterval := time.Duration(networkIntervalFloat * float64(time.Second))
	if networkInterval < 100*time.Millisecond {
		networkInterval = time.Second
		m.logger.Warn("网络监控间隔太小，使用默认值", zap.Duration("interval", networkInterval))
	}

	// 启动网络监控
	m.NetworkMonitor = NewNetworkMonitor(m.logger, networkInterval, m.runMode)
	m.NetworkMonitor.Start()

	// 获取进程监控配置
	processIntervalFloat := viper.GetFloat64("monitor.process.interval")
	processInterval := time.Duration(processIntervalFloat * float64(time.Second))
	if processInterval < 100*time.Millisecond {
		processInterval = time.Second
		m.logger.Warn("进程监控间隔太小，使用默认值", zap.Duration("interval", processInterval))
	}

	// 启动进程监控
	m.ProcessMonitor = NewProcessMonitor(m.logger, processInterval, m.runMode)
	m.ProcessMonitor.Start()

	// 启动系统资源监控
	m.SystemMonitor = NewSystemMonitor(m.logger, sysInterval, diskPaths, m.runMode)
	m.SystemMonitor.Start()

	// 启动硬件信息监控
	m.HardwareMonitor = NewHardwareMonitor(m.logger, hwInterval, hwDiskPaths, m.runMode)
	m.HardwareMonitor.Start()

	// 启动监控协程
	go m.monitor()

	return nil
}

func (m *Monitor) Stop() {
	close(m.stopChan)
	if m.TCPMonitor != nil {
		m.TCPMonitor.Stop()
	}
	if m.SystemMonitor != nil {
		m.SystemMonitor.Stop()
	}
	if m.HardwareMonitor != nil {
		m.HardwareMonitor.Stop()
	}
	if m.HeartbeatMonitor != nil {
		m.HeartbeatMonitor.Stop()
	}
	if m.NetworkMonitor != nil {
		m.NetworkMonitor.Stop()
	}
	if m.ProcessMonitor != nil {
		m.ProcessMonitor.Stop()
	}
	if m.ServerMonitor != nil {
		m.ServerMonitor.Stop()
	}
}

func (m *Monitor) monitor() {
	cmd := exec.Command("tail", "-f", m.logFile)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.logger.Error("创建输出管道失败", zap.Error(err))
		return
	}

	if err := cmd.Start(); err != nil {
		m.logger.Error("启动 tail 命令失败", zap.Error(err))
		return
	}

	// 确保在退出时关闭命令
	defer func() {
		if err := cmd.Process.Kill(); err != nil {
			m.logger.Error("关闭 tail 命令失败", zap.Error(err))
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for {
		select {
		case <-m.stopChan:
			return
		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					m.logger.Error("扫描日志失败", zap.Error(err))
				}
				return
			}
			m.processLine(scanner.Text())
		}
	}
}

// isRecentLogout 检查是否是最近的登出事件
func isRecentLogout(username, ip, port string) bool {
	key := makeLoginKey(username, ip, port)

	logoutRecordMutex.RLock()
	lastLogout, exists := logoutRecords[key]
	logoutRecordMutex.RUnlock()

	if !exists {
		return false
	}

	// 如果在去重时间窗口内有相同的登出事件，则认为是重复的
	return time.Since(lastLogout) < logoutDeduplicationWindow
}

// recordLogout 记录登出事件
func recordLogout(username, ip, port string) {
	key := makeLoginKey(username, ip, port)

	logoutRecordMutex.Lock()
	logoutRecords[key] = time.Now()
	logoutRecordMutex.Unlock()

	// 启动一个 goroutine 在一定时间后清理这条记录
	go func() {
		time.Sleep(logoutDeduplicationWindow)
		logoutRecordMutex.Lock()
		delete(logoutRecords, key)
		logoutRecordMutex.Unlock()
	}()
}

// processLine 处理单行日志内容，检测登录和登出事件
// 参数：
//   - line: 日志行内容
//   - serverInfo: 服务器信息（主机名和IP）
//
// 功能：
//  1. 检测并处理登录事件
//  2. 检测并处理多种类型的登出事件
//  3. 维护登录记录
//  4. 发送登录和登出通知
func (m *Monitor) processLine(line string) {
	// 处理登录事件
	if matches := loginPattern.FindStringSubmatch(line); len(matches) > 0 {
		username := matches[1]
		ip := matches[2]
		port := matches[3]

		// 记录登录信息
		loginRecords[makeLoginKey(username, ip, port)] = types.LoginRecord{
			Username:      username,
			Ip:            ip,
			Port:          port,
			LastLoginTime: time.Now(),
		}

		m.logger.Info("detected login event",
			zap.String("username", username),
			zap.String("ip", ip),
			zap.String("port", port),
		)

		// 获取当前服务器信息
		serverInfo, err := m.ServerMonitor.getServerInfo()
		if err != nil {
			m.logger.Error("获取服务器信息失败", zap.Error(err))
			return
		}

		// 发布登录事件
		m.eventBus.Publish(types.Event{
			Type:       types.TypeLogin,
			Username:   username,
			IP:         ip,
			Port:       port,
			Timestamp:  time.Now(),
			ServerInfo: serverInfo,
		})
		return
	}

	// 处理登出事件
	for _, pattern := range logoutPatterns {
		if matches := pattern.FindStringSubmatch(line); len(matches) > 0 {
			var username, ip, port string

			switch {
			case len(matches) == 4: // Disconnected from user root 192.168.1.1 port 55030
				username = matches[1]
				ip = matches[2]
				port = matches[3]

			case len(matches) == 3 && strings.Contains(line, "Received disconnect"): // Received disconnect
				ip = matches[1]
				port = matches[2]
				// 尝试根据 IP 和端口查找用户名
				for _, record := range loginRecords {
					if record.Ip == ip && record.Port == port {
						username = record.Username
						break
					}
				}
				if username == "" {
					username = "未知用户"
				}

			case len(matches) == 2: // session closed
				username = matches[1]
				// 尝试根据用户名查找最近的登录记录
				for _, record := range loginRecords {
					if record.Username == username {
						ip = record.Ip
						port = record.Port
						break
					}
				}
				if ip == "" {
					ip = "未知IP"
					port = "未知端口"
				}
			}

			// 检查是否是重复的登出事件
			if isRecentLogout(username, ip, port) {
				m.logger.Debug("skipped duplicate logout event",
					zap.String("username", username),
					zap.String("ip", ip),
					zap.String("port", port),
				)
				return
			}

			// 记录这次登出事件
			recordLogout(username, ip, port)

			m.logger.Info("detected logout event",
				zap.String("username", username),
				zap.String("ip", ip),
				zap.String("port", port),
			)

			// 获取当前服务器信息
			serverInfo, err := m.ServerMonitor.getServerInfo()
			if err != nil {
				m.logger.Error("获取服务器信息失败", zap.Error(err))
				return
			}

			// 发布登出事件
			m.eventBus.Publish(types.Event{
				Type:       types.TypeLogout,
				Username:   username,
				IP:         ip,
				Port:       port,
				Timestamp:  time.Now(),
				ServerInfo: serverInfo,
			})

			// 清理登录记录
			if username != "未知用户" && ip != "未知IP" {
				delete(loginRecords, makeLoginKey(username, ip, port))
			}
			return
		}
	}
}
