package notify

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/types"
)

// TelegramNotifier Telegram 通知器
type TelegramNotifier struct {
	*BaseNotifier
	botToken string
	chatID   string
	logger   *zap.Logger
}

// NewTelegramNotifier 创建新的 Telegram 通知器
func NewTelegramNotifier(botToken string, chatID string, logger *zap.Logger) *TelegramNotifier {
	return &TelegramNotifier{
		BaseNotifier: NewBaseNotifier("电报", "Telegram"),
		botToken:     botToken,
		chatID:       chatID,
		logger:       logger,
	}
}

// SendLoginNotification 发送登录通知
func (n *TelegramNotifier) SendLoginNotification(username, ip string, loginTime time.Time, serverInfo *types.ServerInfo) error {
	text := fmt.Sprintf("🔐 用户登录通知\n\n"+
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

	return n.sendMessage(text)
}

// SendLogoutNotification 发送登出通知
func (n *TelegramNotifier) SendLogoutNotification(username, ip string, logoutTime time.Time, serverInfo *types.ServerInfo) error {
	text := fmt.Sprintf("🚪 用户登出通知\n\n"+
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

	return n.sendMessage(text)
}

// sendMessage 发送文本消息
func (n *TelegramNotifier) sendMessage(text string) error {
	// 构建 Telegram Bot API URL
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)

	// 准备请求参数
	params := url.Values{}
	params.Set("chat_id", n.chatID)
	params.Set("text", text)
	params.Set("parse_mode", "HTML")

	// 记录发送请求
	n.logger.Debug("准备发送 Telegram 消息",
		zap.String("chat_id", n.chatID),
		zap.String("text", text),
	)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			n.logger.Error("关闭响应体失败", zap.Error(err))
		}
	}()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API 返回错误状态码: %d", resp.StatusCode)
	}

	n.logger.Debug("Telegram 消息发送成功",
		zap.Int("status_code", resp.StatusCode),
	)

	return nil
}

// sendTestMessage 发送测试消息以验证配置
func (n *TelegramNotifier) sendTestMessage() error {
	text := "🔔 通知服务测试\n\n" +
		"这是一条测试消息，用于验证 Telegram Bot 是否正常工作。"

	return n.sendMessage(text)
}
