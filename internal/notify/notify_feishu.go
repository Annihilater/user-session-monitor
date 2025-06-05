package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/types"
)

// FeishuNotifier 飞书通知器
type FeishuNotifier struct {
	*BaseNotifier
	webhookURL string
	logger     *zap.Logger
}

// NewFeishuNotifier 创建新的飞书通知器
func NewFeishuNotifier(webhookURL string, logger *zap.Logger) *FeishuNotifier {
	return &FeishuNotifier{
		BaseNotifier: NewBaseNotifier(),
		webhookURL:   webhookURL,
		logger:       logger,
	}
}

// Start 启动飞书通知器
func (n *FeishuNotifier) Start(eventChan <-chan types.Event) {
	go func() {
		for {
			select {
			case <-n.stopChan:
				return
			case evt := <-eventChan:
				switch evt.Type {
				case types.TypeLogin:
					if err := n.SendLoginNotification(evt.Username, evt.IP, evt.Timestamp, evt.ServerInfo); err != nil {
						n.logger.Error("发送登录通知失败", zap.Error(err))
					}
				case types.TypeLogout:
					if err := n.SendLogoutNotification(evt.Username, evt.IP, evt.Timestamp, evt.ServerInfo); err != nil {
						n.logger.Error("发送登出通知失败", zap.Error(err))
					}
				}
			}
		}
	}()
}

// SendLoginNotification 发送登录通知
func (n *FeishuNotifier) SendLoginNotification(username, address string, timestamp time.Time, serverInfo *types.ServerInfo) error {
	msg := types.NotifyMessage{
		MsgType: "interactive",
		Content: map[string]interface{}{
			"config": map[string]interface{}{
				"wide_screen_mode": true,
			},
			"elements": []map[string]interface{}{
				{
					"tag": "div",
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**用户**: %s\n**来源**: %s\n**时间**: %s", username, address, timestamp.Format("2006-01-02 15:04:05")),
					},
				},
				{
					"tag": "div",
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**主机名**: %s\n**IP**: %s\n**系统**: %s", serverInfo.Hostname, serverInfo.IP, serverInfo.OSType),
					},
				},
			},
			"header": map[string]interface{}{
				"template": "blue",
				"title": map[string]interface{}{
					"content": "🔐 用户登录通知",
					"tag":     "plain_text",
				},
			},
		},
	}
	return n.sendMessage(msg)
}

// SendLogoutNotification 发送登出通知
func (n *FeishuNotifier) SendLogoutNotification(username, address string, timestamp time.Time, serverInfo *types.ServerInfo) error {
	msg := types.NotifyMessage{
		MsgType: "interactive",
		Content: map[string]interface{}{
			"config": map[string]interface{}{
				"wide_screen_mode": true,
			},
			"elements": []map[string]interface{}{
				{
					"tag": "div",
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**用户**: %s\n**来源**: %s\n**时间**: %s", username, address, timestamp.Format("2006-01-02 15:04:05")),
					},
				},
				{
					"tag": "div",
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**主机名**: %s\n**IP**: %s\n**系统**: %s", serverInfo.Hostname, serverInfo.IP, serverInfo.OSType),
					},
				},
			},
			"header": map[string]interface{}{
				"template": "red",
				"title": map[string]interface{}{
					"content": "🚪 用户登出通知",
					"tag":     "plain_text",
				},
			},
		},
	}
	return n.sendMessage(msg)
}

// sendMessage 发送消息到飞书
func (n *FeishuNotifier) sendMessage(msg types.NotifyMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	n.logger.Debug("准备发送飞书消息",
		zap.String("webhook_url", n.webhookURL),
		zap.String("payload", string(payload)),
	)

	resp, err := http.Post(n.webhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	// 记录响应详情
	n.logger.Debug("收到飞书响应",
		zap.Int("status_code", resp.StatusCode),
		zap.String("response", string(body)),
	)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("请求失败，状态码: %d，响应内容: %s", resp.StatusCode, string(body))
	}

	// 解析飞书响应
	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("解析响应失败: %v, 原始响应: %s", err, string(body))
	}

	// 检查飞书返回的状态码
	if response.Code != 0 {
		return fmt.Errorf("飞书API返回错误: code=%d, msg=%s", response.Code, response.Msg)
	}

	return nil
}

// sendTestMessage 发送测试消息以验证 webhook URL
func (n *FeishuNotifier) sendTestMessage() error {
	msg := types.NotifyMessage{
		MsgType: "interactive",
		Content: map[string]interface{}{
			"config": map[string]interface{}{
				"wide_screen_mode": true,
			},
			"elements": []map[string]interface{}{
				{
					"tag": "div",
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": "**测试消息**\n服务启动时的 webhook 验证",
					},
				},
			},
			"header": map[string]interface{}{
				"template": "blue",
				"title": map[string]interface{}{
					"content": "🔔 通知服务测试",
					"tag":     "plain_text",
				},
			},
		},
	}
	return n.sendMessage(msg)
}
