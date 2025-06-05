package notify

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/types"
)

// EmailNotifier 邮件通知器
type EmailNotifier struct {
	*BaseNotifier
	host     string
	port     string
	username string
	password string
	from     string
	to       []string
	logger   *zap.Logger
}

// NewEmailNotifier 创建新的邮件通知器
func NewEmailNotifier(
	host string,
	port string,
	username string,
	password string,
	from string,
	to []string,
	logger *zap.Logger,
) *EmailNotifier {
	return &EmailNotifier{
		BaseNotifier: NewBaseNotifier("邮件", "Email"),
		host:         host,
		port:         port,
		username:     username,
		password:     password,
		from:         from,
		to:           to,
		logger:       logger,
	}
}

// SendLoginNotification 发送登录通知
func (n *EmailNotifier) SendLoginNotification(username, ip string, loginTime time.Time, serverInfo *types.ServerInfo) error {
	subject := "🔐 用户登录通知"
	body := fmt.Sprintf("用户登录通知\n\n"+
		"用户: %s\n"+
		"来源: %s\n"+
		"时间: %s\n\n"+
		"主机名: %s\n"+
		"IP: %s\n"+
		"系统: %s",
		username, ip,
		loginTime.Format("2006-01-02 15:04:05"),
		serverInfo.Hostname,
		serverInfo.IP,
		serverInfo.OSType,
	)

	return n.sendEmail(subject, body)
}

// SendLogoutNotification 发送登出通知
func (n *EmailNotifier) SendLogoutNotification(username, ip string, logoutTime time.Time, serverInfo *types.ServerInfo) error {
	subject := "🚪 用户登出通知"
	body := fmt.Sprintf("用户登出通知\n\n"+
		"用户: %s\n"+
		"来源: %s\n"+
		"时间: %s\n\n"+
		"主机名: %s\n"+
		"IP: %s\n"+
		"系统: %s",
		username, ip,
		logoutTime.Format("2006-01-02 15:04:05"),
		serverInfo.Hostname,
		serverInfo.IP,
		serverInfo.OSType,
	)

	return n.sendEmail(subject, body)
}

// sendEmail 发送邮件
func (n *EmailNotifier) sendEmail(subject, body string) error {
	// 记录发送请求
	n.logger.Debug("准备发送邮件",
		zap.String("host", n.host),
		zap.String("port", n.port),
		zap.String("from", n.from),
		zap.Strings("to", n.to),
		zap.String("subject", subject),
	)

	// 构建邮件内容
	message := []byte(fmt.Sprintf("To: %s\r\n"+
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

	// 发送邮件
	auth := smtp.PlainAuth("", n.username, n.password, n.host)
	err := smtp.SendMail(
		fmt.Sprintf("%s:%s", n.host, n.port),
		auth,
		n.from,
		n.to,
		message,
	)

	if err != nil {
		return fmt.Errorf("发送邮件失败: %v", err)
	}

	return nil
}

// sendTestMessage 发送测试消息
func (n *EmailNotifier) sendTestMessage() error {
	subject := "🔔 通知服务测试"
	body := "这是一条测试消息，用于验证邮件通知是否正常工作。"
	return n.sendEmail(subject, body)
}
