package monitor

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

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
	loginRecords = make(map[string]loginRecord)

	// 用于存储最近的登出记录，用于去重
	// key 格式：username:ip:port
	// value: 最后一次登出时间
	logoutRecords     = make(map[string]time.Time)
	logoutRecordMutex sync.RWMutex

	// 登出事件的去重时间窗口
	logoutDeduplicationWindow = 5 * time.Second
)

// loginRecord 存储单个登录会话的详细信息
type loginRecord struct {
	username      string    // 用户名
	ip            string    // 登录源 IP
	port          string    // 登录源端口
	lastLoginTime time.Time // 最近一次登录时间
}

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

func getServerInfo() (*types.ServerInfo, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("获取主机名失败: %v", err)
	}

	// 获取非回环IP地址
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("获取网络接口地址失败: %v", err)
	}

	var ip string
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				break
			}
		}
	}

	if ip == "" {
		return nil, fmt.Errorf("未找到有效的IP地址")
	}

	// 获取操作系统类型
	osType, err := detectOSType()
	if err != nil {
		osType = "未知"
	}

	return &types.ServerInfo{
		Hostname: hostname,
		IP:       ip,
		OSType:   osType,
	}, nil
}

type Monitor struct {
	logFile    string
	eventBus   *event.EventBus
	logger     *zap.Logger
	stopChan   chan struct{}
	serverInfo *types.ServerInfo
	TCPMonitor *TCPMonitor // 改为大写，使其可导出
}

func NewMonitor(logFile string, eventBus *event.EventBus, logger *zap.Logger) *Monitor {
	return &Monitor{
		logFile:  logFile,
		eventBus: eventBus,
		logger:   logger,
		stopChan: make(chan struct{}),
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
	file.Close()

	// 获取服务器信息
	serverInfo, err := getServerInfo()
	if err != nil {
		return fmt.Errorf("获取服务器信息失败: %v", err)
	}
	m.serverInfo = serverInfo
	m.logger.Info("服务器信息",
		zap.String("hostname", serverInfo.Hostname),
		zap.String("ip", serverInfo.IP),
		zap.String("os_type", serverInfo.OSType),
		zap.String("log_file", m.logFile),
	)

	// 启动 TCP 监控
	m.TCPMonitor = NewTCPMonitor(m.logger, 1*time.Second) // 每秒监控一次
	m.TCPMonitor.Start()

	// 启动监控协程
	go m.monitor()

	return nil
}

func (m *Monitor) Stop() {
	close(m.stopChan)
	if m.TCPMonitor != nil {
		m.TCPMonitor.Stop()
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
		loginRecords[makeLoginKey(username, ip, port)] = loginRecord{
			username:      username,
			ip:            ip,
			port:          port,
			lastLoginTime: time.Now(),
		}

		m.logger.Info("detected login event",
			zap.String("username", username),
			zap.String("ip", ip),
			zap.String("port", port),
		)

		// 发布登录事件
		m.eventBus.Publish(event.Event{
			Type:       event.EventTypeLogin,
			Username:   username,
			IP:         ip,
			Port:       port,
			Timestamp:  time.Now(),
			ServerInfo: m.serverInfo,
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
					if record.ip == ip && record.port == port {
						username = record.username
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
					if record.username == username {
						ip = record.ip
						port = record.port
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

			// 发布登出事件
			m.eventBus.Publish(event.Event{
				Type:       event.EventTypeLogout,
				Username:   username,
				IP:         ip,
				Port:       port,
				Timestamp:  time.Now(),
				ServerInfo: m.serverInfo,
			})

			// 清理登录记录
			if username != "未知用户" && ip != "未知IP" {
				delete(loginRecords, makeLoginKey(username, ip, port))
			}
			return
		}
	}
}
