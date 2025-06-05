package dingtalk

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/notify/config"
	"github.com/Annihilater/user-session-monitor/internal/notify/notifier"
	"github.com/Annihilater/user-session-monitor/internal/types"
)

// 钉钉消息结构体
type dingTalkMessage struct {
	MsgType string          `json:"msgtype"`
	Text    dingTalkContent `json:"text"`
}

type dingTalkContent struct {
	Content string `json:"content"`
}

// DingTalkNotifier 钉钉通知器
type DingTalkNotifier struct {
	*notifier.BaseNotifier
	webhookURL string
	secret     string
	client     *http.Client
	enabled    bool
}

// validateConfig 验证钉钉配置
func validateConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("配置不能为空")
	}

	if cfg.Type != config.TypeDingTalk {
		return fmt.Errorf("配置类型错误：期望 %s，实际 %s", config.TypeDingTalk, cfg.Type)
	}

	if webhookURL, ok := cfg.Options["webhook_url"]; !ok || webhookURL == "" {
		return fmt.Errorf("webhook_url 不能为空")
	}

	return nil
}

// NewDingTalkNotifier 创建新的钉钉通知器
func NewDingTalkNotifier(cfg *config.Config, logger *zap.Logger) (notifier.Notifier, error) {
	// 验证配置
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	// 创建通知器
	n := &DingTalkNotifier{
		BaseNotifier: notifier.NewBaseNotifier("钉钉", "DingTalk", cfg.Timeout, logger),
		webhookURL:   cfg.Options["webhook_url"],
		secret:       cfg.Options["secret"],
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		enabled: false,
	}

	return n, nil
}

// Initialize 初始化通知器
func (n *DingTalkNotifier) Initialize() error {
	return n.InitializeWithTest(n.sendTestMessage)
}

// IsEnabled 返回通知器是否启用
func (n *DingTalkNotifier) IsEnabled() bool {
	return n.enabled
}

// sendTestMessage 发送测试消息
func (n *DingTalkNotifier) sendTestMessage() error {
	msg := &dingTalkMessage{
		MsgType: "text",
		Text: dingTalkContent{
			Content: "钉钉通知器测试消息",
		},
	}

	if err := n.sendMessage(msg); err != nil {
		return err
	}

	n.enabled = true
	return nil
}

// SendLoginNotification 发送登录通知
func (n *DingTalkNotifier) SendLoginNotification(username, ip string, timestamp time.Time, serverInfo *types.ServerInfo) error {
	msg := &dingTalkMessage{
		MsgType: "text",
		Text: dingTalkContent{
			Content: fmt.Sprintf(
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
func (n *DingTalkNotifier) SendLogoutNotification(username, ip string, timestamp time.Time, serverInfo *types.ServerInfo) error {
	msg := &dingTalkMessage{
		MsgType: "text",
		Text: dingTalkContent{
			Content: fmt.Sprintf(
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

// sendMessage 发送消息到钉钉
func (n *DingTalkNotifier) sendMessage(msg *dingTalkMessage) error {
	// 将消息转换为 JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("消息序列化失败：%v", err)
	}

	// 生成签名URL
	webhookURL := n.webhookURL
	if n.secret != "" {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		sign := n.generateSign(timestamp)
		webhookURL = fmt.Sprintf("%s&timestamp=%s&sign=%s", n.webhookURL, timestamp, url.QueryEscape(sign))
	}

	// 创建请求
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
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

// generateSign 生成签名
func (n *DingTalkNotifier) generateSign(timestamp string) string {
	stringToSign := fmt.Sprintf("%s\n%s", timestamp, n.secret)
	h := hmac.New(sha256.New, []byte(n.secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
