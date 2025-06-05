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
		BaseNotifier: NewBaseNotifier("飞书", "Feishu"),
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
func (n *FeishuNotifier) SendLoginNotification(username, ip string, loginTime time.Time, serverInfo *types.ServerInfo) error {
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
func (n *FeishuNotifier) SendLogoutNotification(username, ip string, logoutTime time.Time, serverInfo *types.ServerInfo) error {
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
func (n *FeishuNotifier) sendMessage(text string) error {
	message := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]interface{}{
			"text": text,
		},
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	n.logger.Debug("准备发送飞书消息",
		zap.String("webhook_url", n.webhookURL),
		zap.String("payload", string(jsonData)),
	)

	// 创建请求
	req, err := http.NewRequest("POST", n.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			n.logger.Error("关闭响应体失败", zap.Error(err))
		}
	}()

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

	// 解析响应
	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("解析响应失败: %v, 原始响应: %s", err, string(body))
	}

	// 检查响应状态
	if response.Code != 0 {
		return fmt.Errorf("飞书API返回错误: code=%d, msg=%s", response.Code, response.Msg)
	}

	return nil
}

// sendTestMessage 发送测试消息以验证 webhook URL
func (n *FeishuNotifier) sendTestMessage() error {
	text := "🔔 通知服务测试\n\n" +
		"这是一条测试消息，用于验证飞书机器人是否正常工作。"

	return n.sendMessage(text)
}
