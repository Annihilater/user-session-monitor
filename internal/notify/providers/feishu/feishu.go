package feishu

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

// 飞书消息结构体
type feishuMessage struct {
	MsgType string        `json:"msg_type"`
	Content feishuContent `json:"content"`
}

type feishuContent struct {
	Text string `json:"text"`
}

// FeishuNotifier 飞书通知器
type FeishuNotifier struct {
	*notifier.BaseNotifier
	webhookURL string
	client     *http.Client
	enabled    bool
}

// validateConfig 验证飞书配置
func validateConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("配置不能为空")
	}

	if cfg.Type != config.TypeFeishu {
		return fmt.Errorf("配置类型错误：期望 %s，实际 %s", config.TypeFeishu, cfg.Type)
	}

	if webhookURL, ok := cfg.Options["webhook_url"]; !ok || webhookURL == "" {
		return fmt.Errorf("webhook_url 不能为空")
	}

	return nil
}

// NewFeishuNotifier 创建新的飞书通知器
func NewFeishuNotifier(cfg *config.Config, logger *zap.Logger) (notifier.Notifier, error) {
	// 验证配置
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	// 创建通知器
	n := &FeishuNotifier{
		BaseNotifier: notifier.NewBaseNotifier("飞书", "Feishu", cfg.Timeout, logger),
		webhookURL:   cfg.Options["webhook_url"],
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		enabled: false,
	}

	return n, nil
}

// Initialize 初始化通知器
func (n *FeishuNotifier) Initialize() error {
	return n.InitializeWithTest(n.sendTestMessage)
}

// IsEnabled 返回通知器是否启用
func (n *FeishuNotifier) IsEnabled() bool {
	return n.enabled
}

// sendTestMessage 发送测试消息
func (n *FeishuNotifier) sendTestMessage() error {
	msg := &feishuMessage{
		MsgType: "text",
		Content: feishuContent{
			Text: "飞书通知器测试消息",
		},
	}

	if err := n.sendMessage(msg); err != nil {
		return err
	}

	n.enabled = true
	return nil
}

// SendLoginNotification 发送登录通知
func (n *FeishuNotifier) SendLoginNotification(username, ip string, timestamp time.Time, serverInfo *types.ServerInfo) error {
	msg := &feishuMessage{
		MsgType: "text",
		Content: feishuContent{
			Text: fmt.Sprintf(
				"🔔 用户登录通知\n时间：%s\n用户：%s\n来源IP：%s\n服务器：%s (%s)",
				timestamp.Format("2006-01-02 15:04:05"),
				username,
				ip,
				serverInfo.Hostname,
				serverInfo.IP,
			),
		},
	}
	return n.sendMessage(msg)
}

// SendLogoutNotification 发送登出通知
func (n *FeishuNotifier) SendLogoutNotification(username, ip string, timestamp time.Time, serverInfo *types.ServerInfo) error {
	msg := &feishuMessage{
		MsgType: "text",
		Content: feishuContent{
			Text: fmt.Sprintf(
				"🔔 用户登出通知\n时间：%s\n用户：%s\n来源IP：%s\n服务器：%s (%s)",
				timestamp.Format("2006-01-02 15:04:05"),
				username,
				ip,
				serverInfo.Hostname,
				serverInfo.IP,
			),
		},
	}
	return n.sendMessage(msg)
}

// sendMessage 发送消息到飞书
func (n *FeishuNotifier) sendMessage(msg *feishuMessage) error {
	// 将消息转换为 JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("消息序列化失败：%v", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", n.webhookURL, bytes.NewBuffer(jsonData))
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
