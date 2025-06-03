package monitor

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/feishu"
)

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
	logger   *zap.Logger
}

func NewMonitor(logFile string, notifier *feishu.Notifier, logger *zap.Logger) *Monitor {
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
	m.logger.Info("server info",
		zap.String("hostname", serverInfo.Hostname),
		zap.String("ip", serverInfo.IP),
	)

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
//
// 功能：
//  1. 检测并处理登录事件
//  2. 检测并处理多种类型的登出事件
//  3. 维护登录记录
//  4. 发送登录和登出通知
func (m *Monitor) processLine(line string, serverInfo *feishu.ServerInfo) {
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

		if err := m.notifier.SendLoginNotification(
			username,
			fmt.Sprintf("%s:%s", ip, port),
			time.Now(),
			serverInfo,
		); err != nil {
			m.logger.Error("failed to send login notification",
				zap.Error(err),
				zap.String("username", username),
				zap.String("ip", ip),
				zap.String("port", port),
			)
		}
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

			m.logger.Info("detected logout event",
				zap.String("username", username),
				zap.String("ip", ip),
				zap.String("port", port),
			)

			// 发送登出通知
			if err := m.notifier.SendLogoutNotification(
				fmt.Sprintf("%s (IP: %s:%s)", username, ip, port),
				time.Now(),
				serverInfo,
			); err != nil {
				m.logger.Error("failed to send logout notification",
					zap.Error(err),
					zap.String("username", username),
					zap.String("ip", ip),
					zap.String("port", port),
				)
			}

			// 清理登录记录
			if username != "未知用户" && ip != "未知IP" {
				delete(loginRecords, makeLoginKey(username, ip, port))
			}
			return
		}
	}
}
