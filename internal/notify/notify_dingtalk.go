package notify

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/types"
)

// DingTalkNotifier 钉钉通知器
type DingTalkNotifier struct {
	*BaseNotifier
	webhookURL string
	secret     string
	logger     *zap.Logger
}

// NewDingTalkNotifier 创建新的钉钉通知器
func NewDingTalkNotifier(webhookURL string, secret string, logger *zap.Logger) *DingTalkNotifier {
	return &DingTalkNotifier{
		BaseNotifier: NewBaseNotifier(),
		webhookURL:   webhookURL,
		secret:       secret,
		logger:       logger,
	}
}

// Start 启动钉钉通知器
func (n *DingTalkNotifier) Start(eventChan <-chan types.Event) {
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
func (n *DingTalkNotifier) SendLoginNotification(username, ip string, loginTime time.Time, serverInfo *types.ServerInfo) error {
	title := "⚠️ 用户登录通知"
	text := fmt.Sprintf("### ⚠️ 用户登录通知\n\n"+
		"**用户名**: %s\n\n"+
		"**登录IP**: %s\n\n"+
		"**登录时间**: %s\n\n"+
		"**服务器信息**:\n\n"+
		"- 主机名: %s\n"+
		"- 服务器IP: %s\n"+
		"- 系统类型: %s\n",
		username, ip,
		loginTime.Format("2006-01-02 15:04:05"),
		serverInfo.Hostname,
		serverInfo.IP,
		serverInfo.OSType,
	)

	return n.sendMarkdown(title, text)
}

// SendLogoutNotification 发送登出通知
func (n *DingTalkNotifier) SendLogoutNotification(username, ip string, logoutTime time.Time, serverInfo *types.ServerInfo) error {
	title := "🔔 用户登出通知"
	text := fmt.Sprintf("### 🔔 用户登出通知\n\n"+
		"**用户名**: %s\n\n"+
		"**登出IP**: %s\n\n"+
		"**登出时间**: %s\n\n"+
		"**服务器信息**:\n\n"+
		"- 主机名: %s\n"+
		"- 服务器IP: %s\n"+
		"- 系统类型: %s\n",
		username, ip,
		logoutTime.Format("2006-01-02 15:04:05"),
		serverInfo.Hostname,
		serverInfo.IP,
		serverInfo.OSType,
	)

	return n.sendMarkdown(title, text)
}

// sendMarkdown 发送 Markdown 格式消息
func (n *DingTalkNotifier) sendMarkdown(title, text string) error {
	message := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  text,
		},
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	// 如果设置了加签密钥，则生成签名
	webhookURL := n.webhookURL
	if n.secret != "" {
		timestamp := time.Now().UnixMilli()
		sign := n.generateSign(timestamp)
		webhookURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", n.webhookURL, timestamp, url.QueryEscape(sign))
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	return nil
}

// generateSign 生成钉钉签名
func (n *DingTalkNotifier) generateSign(timestamp int64) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, n.secret)
	h := hmac.New(sha256.New, []byte(n.secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// sendTestMessage 发送测试消息以验证 webhook URL
func (n *DingTalkNotifier) sendTestMessage() error {
	title := "🔔 通知服务测试"
	text := "### 🔔 通知服务测试\n\n" +
		"**测试消息**\n\n" +
		"服务启动时的 webhook 验证"

	return n.sendMarkdown(title, text)
}
