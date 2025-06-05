package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/notify/config"
	"github.com/Annihilater/user-session-monitor/internal/notify/notifier"
	"github.com/Annihilater/user-session-monitor/internal/types"
)

// Telegram API 相关常量
const (
	telegramAPIBaseURL = "https://api.telegram.org/bot%s/sendMessage"
)

// Telegram 消息结构体
type telegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// TelegramNotifier Telegram 通知器
type TelegramNotifier struct {
	*notifier.BaseNotifier
	botToken string
	chatID   string
	client   *http.Client
	enabled  bool
}

// validateConfig 验证 Telegram 配置
func validateConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("配置不能为空")
	}

	if cfg.Type != config.TypeTelegram {
		return fmt.Errorf("配置类型错误：期望 %s，实际 %s", config.TypeTelegram, cfg.Type)
	}

	if botToken, ok := cfg.Options["bot_token"]; !ok || botToken == "" {
		return fmt.Errorf("bot_token 不能为空")
	}

	if chatID, ok := cfg.Options["chat_id"]; !ok || chatID == "" {
		return fmt.Errorf("chat_id 不能为空")
	}

	return nil
}

// NewTelegramNotifier 创建新的 Telegram 通知器
func NewTelegramNotifier(cfg *config.Config, logger *zap.Logger) (notifier.Notifier, error) {
	// 验证配置
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	// 创建通知器
	n := &TelegramNotifier{
		BaseNotifier: notifier.NewBaseNotifier("Telegram", "Telegram", cfg.Timeout, logger),
		botToken:     cfg.Options["bot_token"],
		chatID:       cfg.Options["chat_id"],
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		enabled: false,
	}

	return n, nil
}

// Initialize 初始化通知器
func (n *TelegramNotifier) Initialize() error {
	return n.InitializeWithTest(n.sendTestMessage)
}

// IsEnabled 返回通知器是否启用
func (n *TelegramNotifier) IsEnabled() bool {
	return n.enabled
}

// sendTestMessage 发送测试消息
func (n *TelegramNotifier) sendTestMessage() error {
	msg := &telegramMessage{
		ChatID: n.chatID,
		Text:   "Telegram 通知器测试消息",
	}

	if err := n.sendMessage(msg); err != nil {
		return err
	}

	n.enabled = true
	return nil
}

// SendLoginNotification 发送登录通知
func (n *TelegramNotifier) SendLoginNotification(username, ip string, timestamp time.Time, serverInfo *types.ServerInfo) error {
	msg := &telegramMessage{
		ChatID: n.chatID,
		Text: fmt.Sprintf(
			"🔔 用户登录通知\n时间：%s\n用户：%s\n来源IP：%s\n服务器：%s (%s)",
			timestamp.Format("2006-01-02 15:04:05"),
			username,
			ip,
			serverInfo.Hostname,
			serverInfo.IP,
		),
	}
	return n.sendMessage(msg)
}

// SendLogoutNotification 发送登出通知
func (n *TelegramNotifier) SendLogoutNotification(username, ip string, timestamp time.Time, serverInfo *types.ServerInfo) error {
	msg := &telegramMessage{
		ChatID: n.chatID,
		Text: fmt.Sprintf(
			"🔔 用户登出通知\n时间：%s\n用户：%s\n来源IP：%s\n服务器：%s (%s)",
			timestamp.Format("2006-01-02 15:04:05"),
			username,
			ip,
			serverInfo.Hostname,
			serverInfo.IP,
		),
	}
	return n.sendMessage(msg)
}

// sendMessage 发送消息到 Telegram
func (n *TelegramNotifier) sendMessage(msg *telegramMessage) error {
	// 将消息转换为 JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("消息序列化失败：%v", err)
	}

	// 创建请求
	apiURL := fmt.Sprintf(telegramAPIBaseURL, n.botToken)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败：%v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), n.client.Timeout)
	defer cancel()
	req = req.WithContext(ctx)

	// 发送请求
	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败：%v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			n.BaseNotifier.GetLogger().Error("关闭响应体失败", zap.Error(closeErr))
		}
	}()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("请求失败，状态码：%d", resp.StatusCode)
	}

	return nil
}
