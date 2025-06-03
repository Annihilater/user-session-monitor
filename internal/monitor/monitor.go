package monitor

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"time"

	"github.com/ziji/user-session-monitor/internal/feishu"
)

var (
	loginPattern  = regexp.MustCompile(`(?m)sshd\[\d+\]: Accepted (?:password|publickey) for (\w+) from ([\d\.]+)`)
	logoutPattern = regexp.MustCompile(`(?m)sshd\[\d+\]: Received disconnect from ([\d\.]+) .* Disconnected by user`)
)

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
		m.processLine(line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %v", err)
	}

	return nil
}

func (m *Monitor) processLine(line string) {
	// 处理登录事件
	if matches := loginPattern.FindStringSubmatch(line); len(matches) > 0 {
		username := matches[1]
		ip := matches[2]
		m.logger.Printf("detected login event: username=%s, ip=%s", username, ip)

		if err := m.notifier.SendLoginNotification(username, ip, time.Now()); err != nil {
			m.logger.Printf("failed to send login notification: %v", err)
		}
	}

	// 处理登出事件
	if matches := logoutPattern.FindStringSubmatch(line); len(matches) > 0 {
		ip := matches[1]
		// 注意：登出事件中可能无法直接获取用户名
		m.logger.Printf("detected logout event: ip=%s", ip)

		// 这里我们只发送 IP 信息，因为日志中可能没有用户名
		if err := m.notifier.SendLogoutNotification(fmt.Sprintf("IP: %s", ip), time.Now()); err != nil {
			m.logger.Printf("failed to send logout notification: %v", err)
		}
	}
}
