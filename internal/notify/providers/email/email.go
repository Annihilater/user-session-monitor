package email

import (
	"context"
	"fmt"
	"strings"
	"time"

	"net/smtp"

	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/notify/config"
	"github.com/Annihilater/user-session-monitor/internal/notify/notifier"
	"github.com/Annihilater/user-session-monitor/internal/types"
)

// EmailNotifier 邮件通知器
type EmailNotifier struct {
	*notifier.BaseNotifier
	host     string
	port     string
	username string
	password string
	from     string
	to       []string
	logger   *zap.Logger
	enabled  bool
	timeout  time.Duration
}

// validateConfig 验证邮件配置
func validateConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("配置不能为空")
	}

	if cfg.Type != config.TypeEmail {
		return fmt.Errorf("配置类型错误：期望 %s，实际 %s", config.TypeEmail, cfg.Type)
	}

	required := []string{"host", "port", "username", "password", "from", "to"}
	for _, field := range required {
		if value, ok := cfg.Options[field]; !ok || value == "" {
			return fmt.Errorf("%s 不能为空", field)
		}
	}

	return nil
}

// NewEmailNotifier 创建新的邮件通知器
func NewEmailNotifier(cfg *config.Config, logger *zap.Logger) (notifier.Notifier, error) {
	// 验证配置
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	// 创建通知器
	n := &EmailNotifier{
		BaseNotifier: notifier.NewBaseNotifier("邮件", "Email", cfg.Timeout, logger),
		host:         cfg.Options["host"],
		port:         cfg.Options["port"],
		username:     cfg.Options["username"],
		password:     cfg.Options["password"],
		from:         cfg.Options["from"],
		to:           strings.Split(cfg.Options["to"], ","),
		enabled:      false,
		timeout:      cfg.Timeout,
	}

	return n, nil
}

// Initialize 初始化通知器
func (n *EmailNotifier) Initialize() error {
	return n.InitializeWithTest(n.sendTestMessage)
}

// IsEnabled 返回通知器是否启用
func (n *EmailNotifier) IsEnabled() bool {
	return n.enabled
}

// sendTestMessage 发送测试消息
func (n *EmailNotifier) sendTestMessage() error {
	subject := "邮件通知器测试消息"
	body := "这是一条测试消息，用于验证邮件通知器是否正常工作。"

	if err := n.sendEmail(subject, body); err != nil {
		return err
	}

	n.enabled = true
	return nil
}

// SendLoginNotification 发送登录通知
func (n *EmailNotifier) SendLoginNotification(username, ip string, timestamp time.Time, serverInfo *types.ServerInfo) error {
	subject := fmt.Sprintf("用户登录通知 - %s", username)
	body := fmt.Sprintf(
		"🔔 用户登录通知\n时间：%s\n用户：%s\n来源IP：%s\n服务器：%s (%s)",
		timestamp.Format("2006-01-02 15:04:05"),
		username,
		ip,
		serverInfo.Hostname,
		serverInfo.IP,
	)
	return n.sendEmail(subject, body)
}

// SendLogoutNotification 发送登出通知
func (n *EmailNotifier) SendLogoutNotification(username, ip string, timestamp time.Time, serverInfo *types.ServerInfo) error {
	subject := fmt.Sprintf("用户登出通知 - %s", username)
	body := fmt.Sprintf(
		"🔔 用户登出通知\n时间：%s\n用户：%s\n来源IP：%s\n服务器：%s (%s)",
		timestamp.Format("2006-01-02 15:04:05"),
		username,
		ip,
		serverInfo.Hostname,
		serverInfo.IP,
	)
	return n.sendEmail(subject, body)
}

// sendEmail 发送邮件
func (n *EmailNotifier) sendEmail(subject, body string) error {
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), n.timeout)
	defer cancel()

	// 在协程中发送邮件
	errChan := make(chan error, 1)
	go func() {
		errChan <- n.doSendEmail(subject, body)
	}()

	// 等待邮件发送完成或超时
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("发送邮件超时（%v）", n.timeout)
	}
}

// doSendEmail 实际发送邮件的函数
func (n *EmailNotifier) doSendEmail(subject, body string) error {
	// 构建邮件内容
	message := []byte(fmt.Sprintf(
		"To: %s\r\n"+
			"From: %s\r\n"+
			"Subject: %s\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n"+
			"\r\n"+
			"%s",
		strings.Join(n.to, ","),
		n.from,
		subject,
		body,
	))

	// 创建 SMTP 客户端
	auth := smtp.PlainAuth("", n.username, n.password, n.host)
	addr := fmt.Sprintf("%s:%s", n.host, n.port)

	// 发送邮件
	if err := smtp.SendMail(addr, auth, n.from, n.to, message); err != nil {
		return fmt.Errorf("发送邮件失败：%v", err)
	}

	return nil
}
