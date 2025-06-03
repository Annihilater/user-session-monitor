package monitor

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/ziji/user-session-monitor/internal/feishu"
)

var (
	// 登录事件匹配模式
	// 匹配示例：
	// sshd[12345]: Accepted password for userA from 192.168.1.100 port 22 ssh2
	// sshd[12345]: Accepted publickey for userB from 10.0.0.100 port 22 ssh2
	loginPattern = regexp.MustCompile(`(?m)sshd\[\d+\]: Accepted (?:password|publickey) for (\w+) from ([\d\.]+)`)

	// 登出事件匹配模式列表
	logoutPatterns = []*regexp.Regexp{
		// 1. 用户主动断开连接场景
		// 匹配示例：sshd[12345]: Received disconnect from 192.168.1.100 port 22:11: disconnected by user
		// 常见于：
		// - 使用 exit 命令
		// - 使用 logout 命令
		// - 按 Ctrl + D
		regexp.MustCompile(`(?m)sshd\[\d+\]: Received disconnect from ([\d\.]+) .* Disconnected by user`),

		// 2. 用户断开连接场景（带用户名）
		// 匹配示例：sshd[12345]: User userA from 192.168.1.100 disconnected
		// 常见于：
		// - 客户端正常关闭 SSH 连接
		// - 网络正常断开
		regexp.MustCompile(`(?m)sshd\[\d+\]: User (\w+) from ([\d\.]+) disconnected`),

		// 3. 认证用户关闭连接场景
		// 匹配示例：sshd[12345]: Connection closed by authenticating user userA 192.168.1.100
		// 常见于：
		// - SSH 客户端异常退出
		// - 网络异常断开
		// - 客户端超时断开
		regexp.MustCompile(`(?m)sshd\[\d+\]: Connection closed by authenticating user (\w+) ([\d\.]+)`),

		// 4. PAM 会话关闭场景
		// 匹配示例：sshd[12345]: pam_unix(sshd:session): session closed for user userA
		// 常见于：
		// - 系统强制关闭会话
		// - 会话超时自动登出
		// - 系统关机或重启
		regexp.MustCompile(`(?m)sshd\[\d+\]: pam_unix\(sshd:session\): session closed for user (\w+)`),
	}
)

func getServerInfo() (*feishu.ServerInfo, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("get hostname failed: %v", err)
	}

	// 获取非回环IP地址
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("get interface addresses failed: %v", err)
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return &feishu.ServerInfo{
					Hostname: hostname,
					IP:       ipnet.IP.String(),
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("no valid IP address found")
}

type Monitor struct {
	logFile  string
	notifier *feishu.Notifier
	logger   *log.Logger
}

func NewMonitor(logFile string, notifier *feishu.Notifier, logger *log.Logger) *Monitor {
	return &Monitor{
		logFile:  logFile,
		notifier: notifier,
		logger:   logger,
	}
}

func (m *Monitor) Start() error {
	// 检查日志文件是否存在且可读
	if _, err := os.Stat(m.logFile); os.IsNotExist(err) {
		return fmt.Errorf("log file %s does not exist", m.logFile)
	}

	// 尝试打开文件以验证权限
	file, err := os.Open(m.logFile)
	if err != nil {
		return fmt.Errorf("cannot open log file %s: %v", m.logFile, err)
	}
	file.Close()

	// 获取服务器信息
	serverInfo, err := getServerInfo()
	if err != nil {
		return fmt.Errorf("failed to get server info: %v", err)
	}
	m.logger.Printf("Server Info - Hostname: %s, IP: %s", serverInfo.Hostname, serverInfo.IP)

	cmd := exec.Command("tail", "-f", m.logFile)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start tail command: %v", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		m.processLine(line, serverInfo)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %v", err)
	}

	return nil
}

// processLine 处理单行日志内容，检测登录和登出事件
// 参数：
//   - line: 日志行内容
//   - serverInfo: 服务器信息（主机名和IP）
func (m *Monitor) processLine(line string, serverInfo *feishu.ServerInfo) {
	// 处理登录事件
	// 从日志中提取用户名和IP地址
	if matches := loginPattern.FindStringSubmatch(line); len(matches) > 0 {
		username := matches[1]
		ip := matches[2]
		m.logger.Printf("detected login event: username=%s, ip=%s", username, ip)

		if err := m.notifier.SendLoginNotification(username, ip, time.Now(), serverInfo); err != nil {
			m.logger.Printf("failed to send login notification: %v", err)
		}
		return
	}

	// 处理登出事件
	// 遍历所有登出模式，匹配第一个符合的模式
	for _, pattern := range logoutPatterns {
		if matches := pattern.FindStringSubmatch(line); len(matches) > 0 {
			var username, ip string

			// 根据匹配组的数量来确定信息的提取方式
			switch len(matches) {
			case 2: // 只匹配到一个组（用户名或IP）
				if pattern.String() == logoutPatterns[3].String() {
					// PAM 会话关闭场景：只有用户名
					username = matches[1]
					ip = "未知IP"
				} else {
					// 其他只有IP的场景
					username = "未知用户"
					ip = matches[1]
				}
			case 3: // 匹配到用户名和IP
				username = matches[1]
				ip = matches[2]
			default:
				continue
			}

			m.logger.Printf("detected logout event: username=%s, ip=%s", username, ip)

			// 发送登出通知
			// 将用户名和IP组合在一起显示
			if err := m.notifier.SendLogoutNotification(
				fmt.Sprintf("%s (IP: %s)", username, ip),
				time.Now(),
				serverInfo,
			); err != nil {
				m.logger.Printf("failed to send logout notification: %v", err)
			}
			return
		}
	}
}
