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
	loginPattern = regexp.MustCompile(`(?m)sshd\[\d+\]: Accepted (?:password|publickey) for (\w+) from ([\d\.]+)`)
	// 支持多种登出场景的正则表达式
	logoutPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?m)sshd\[\d+\]: Received disconnect from ([\d\.]+) .* Disconnected by user`),
		regexp.MustCompile(`(?m)sshd\[\d+\]: User (\w+) from ([\d\.]+) disconnected`),
		regexp.MustCompile(`(?m)sshd\[\d+\]: Connection closed by authenticating user (\w+) ([\d\.]+)`),
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

func (m *Monitor) processLine(line string, serverInfo *feishu.ServerInfo) {
	// 处理登录事件
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
	for _, pattern := range logoutPatterns {
		if matches := pattern.FindStringSubmatch(line); len(matches) > 0 {
			var username, ip string

			switch len(matches) {
			case 2: // 只匹配到一个组（用户名或IP）
				if pattern.String() == logoutPatterns[3].String() {
					// 匹配 "session closed for user" 模式
					username = matches[1]
					ip = "未知IP"
				} else {
					// 其他只有IP的模式
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
