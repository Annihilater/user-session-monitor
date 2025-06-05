package notify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/spf13/viper"
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
	enabled  bool // 标记通知器是否可用
	timeout  time.Duration
}

// checkConnection 检查与 SMTP 服务器的连接
func (n *EmailNotifier) checkConnection() error {
	n.logger.Info("检查 SMTP 服务器连接")

	// 使用配置的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), n.timeout)
	defer cancel()

	// 创建一个 Dialer
	var d net.Dialer
	addr := fmt.Sprintf("%s:%s", n.host, n.port)

	// 尝试建立 TCP 连接
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("无法连接到 SMTP 服务器 %s: %v", addr, err)
	}
	defer conn.Close()

	n.logger.Info("SMTP 服务器连接检查成功")
	return nil
}

// sendEmailWithTimeout 带超时的邮件发送
func (n *EmailNotifier) sendEmailWithTimeout(subject, body string) error {
	// 使用配置的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), n.timeout)
	defer cancel()

	// 创建一个错误通道
	errChan := make(chan error, 1)

	// 在新的 goroutine 中执行发送操作
	go func() {
		errChan <- n.doSendEmail(ctx, subject, body)
	}()

	// 等待发送完成或超时
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("发送邮件超时（%v）", n.timeout)
	}
}

// doSendEmail 实际执行邮件发送的方法
func (n *EmailNotifier) doSendEmail(ctx context.Context, subject, body string) error {
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

	n.logger.Info("邮件内容构建完成")

	// 创建一个 Dialer
	var d net.Dialer
	addr := fmt.Sprintf("%s:%s", n.host, n.port)

	// 尝试建立 TCP 连接
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		n.logger.Error("连接 SMTP 服务器失败",
			zap.String("addr", addr),
			zap.Error(err),
		)
		return fmt.Errorf("连接 SMTP 服务器失败: %v", err)
	}
	defer conn.Close()

	// 配置 TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         n.host,
	}

	// 创建 SMTP 客户端
	client, err := smtp.NewClient(conn, n.host)
	if err != nil {
		n.logger.Error("创建 SMTP 客户端失败", zap.Error(err))
		return fmt.Errorf("创建 SMTP 客户端失败: %v", err)
	}
	defer client.Close()

	// 启用 TLS
	if err = client.StartTLS(tlsConfig); err != nil {
		n.logger.Error("启用 TLS 失败", zap.Error(err))
		return fmt.Errorf("启用 TLS 失败: %v", err)
	}

	// 认证
	auth := smtp.PlainAuth("", n.username, n.password, n.host)
	if err = client.Auth(auth); err != nil {
		n.logger.Error("SMTP 认证失败", zap.Error(err))
		return fmt.Errorf("SMTP 认证失败: %v", err)
	}

	// 设置发件人
	if err = client.Mail(n.from); err != nil {
		n.logger.Error("设置发件人失败", zap.Error(err))
		return fmt.Errorf("设置发件人失败: %v", err)
	}

	// 设置收件人
	for _, recipient := range n.to {
		if err = client.Rcpt(recipient); err != nil {
			n.logger.Error("设置收件人失败",
				zap.String("recipient", recipient),
				zap.Error(err),
			)
			return fmt.Errorf("设置收件人失败: %v", err)
		}
	}

	// 发送邮件内容
	w, err := client.Data()
	if err != nil {
		n.logger.Error("准备发送数据失败", zap.Error(err))
		return fmt.Errorf("准备发送数据失败: %v", err)
	}
	defer w.Close()

	if _, err = w.Write(message); err != nil {
		n.logger.Error("写入邮件内容失败", zap.Error(err))
		return fmt.Errorf("写入邮件内容失败: %v", err)
	}

	n.logger.Info("邮件发送成功",
		zap.String("subject", subject),
		zap.Strings("to", n.to),
	)
	return nil
}

// sendEmail 发送邮件的入口方法
func (n *EmailNotifier) sendEmail(subject, body string) error {
	if !n.enabled {
		return fmt.Errorf("邮件通知器未启用")
	}

	// 使用带超时的发送方法
	err := n.sendEmailWithTimeout(subject, body)
	if err != nil {
		if err.Error() == fmt.Sprintf("发送邮件超时（%v）", n.timeout) {
			n.logger.Warn("发送邮件超时，将禁用邮件通知器",
				zap.Duration("timeout", n.timeout),
			)
			n.enabled = false
		}
		return err
	}

	return nil
}

// SendLoginNotification 发送登录通知
func (n *EmailNotifier) SendLoginNotification(username, ip string, loginTime time.Time, serverInfo *types.ServerInfo) error {
	if !n.enabled {
		n.logger.Warn("邮件通知器未启用，跳过发送登录通知")
		return nil
	}

	n.logger.Info("准备发送登录通知",
		zap.String("username", username),
		zap.String("ip", ip),
	)
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
	if !n.enabled {
		n.logger.Warn("邮件通知器未启用，跳过发送登出通知")
		return nil
	}

	n.logger.Info("准备发送登出通知",
		zap.String("username", username),
		zap.String("ip", ip),
	)
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

// sendTestMessage 发送测试消息
func (n *EmailNotifier) sendTestMessage() error {
	if !n.enabled {
		n.logger.Warn("邮件通知器未启用，跳过发送测试消息")
		return nil
	}

	n.logger.Info("准备发送测试消息")
	subject := "🔔 通知服务测试"
	body := "这是一条测试消息，用于验证邮件通知是否正常工作。"
	err := n.sendEmail(subject, body)
	if err != nil {
		n.logger.Error("发送测试消息失败", zap.Error(err))
		return fmt.Errorf("发送测试消息失败: %v", err)
	}
	n.logger.Info("测试消息发送成功")
	return nil
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
	// 从配置文件获取超时时间，默认3秒
	timeoutSeconds := viper.GetFloat64("notify.email.timeout")
	if timeoutSeconds <= 0 {
		timeoutSeconds = 3
	}
	timeout := time.Duration(timeoutSeconds * float64(time.Second))

	notifier := &EmailNotifier{
		BaseNotifier: NewBaseNotifier("邮件", "Email"),
		host:         host,
		port:         port,
		username:     username,
		password:     password,
		from:         from,
		to:           to,
		logger:       logger,
		enabled:      false, // 默认禁用，直到验证成功
		timeout:      timeout,
	}

	// 验证配置
	if err := notifier.validateConfig(); err != nil {
		logger.Error("邮件通知器配置验证失败", zap.Error(err))
		return notifier
	}
	logger.Info("邮件通知器配置验证成功")

	// 检查连接
	if err := notifier.checkConnection(); err != nil {
		logger.Warn("SMTP 服务器连接失败，邮件通知器将被禁用",
			zap.Error(err),
			zap.String("host", host),
			zap.String("port", port),
		)
		return notifier
	}

	// 所有检查都通过，启用通知器
	notifier.enabled = true
	logger.Info("邮件通知器初始化成功并已启用")
	return notifier
}

// validateConfig 验证配置是否有效
func (n *EmailNotifier) validateConfig() error {
	if n.host == "" {
		return fmt.Errorf("SMTP 主机不能为空")
	}
	if n.port == "" {
		return fmt.Errorf("SMTP 端口不能为空")
	}
	if n.username == "" {
		return fmt.Errorf("SMTP 用户名不能为空")
	}
	if n.password == "" {
		return fmt.Errorf("SMTP 密码不能为空")
	}
	if len(n.to) == 0 {
		return fmt.Errorf("收件人列表不能为空")
	}
	return nil
}
