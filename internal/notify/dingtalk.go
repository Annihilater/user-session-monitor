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
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/types"
)

// DingTalkNotifier 钉钉通知器
type DingTalkNotifier struct {
	webhookURL string
	secret     string
	logger     *zap.Logger
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// NewDingTalkNotifier 创建新的钉钉通知器
func NewDingTalkNotifier(webhookURL string, secret string, logger *zap.Logger) *DingTalkNotifier {
	return &DingTalkNotifier{
		webhookURL: webhookURL,
		secret:     secret,
		logger:     logger,
		stopChan:   make(chan struct{}),
	}
}

// Start 启动通知处理器
func (n *DingTalkNotifier) Start(eventChan <-chan types.Event) {
	n.wg.Add(1)
	go n.processEvents(eventChan)
}

// Stop 停止通知处理器
func (n *DingTalkNotifier) Stop() {
	close(n.stopChan)
	n.wg.Wait()
}

// processEvents 处理事件
func (n *DingTalkNotifier) processEvents(eventChan <-chan types.Event) {
	defer n.wg.Done()

	for {
		select {
		case <-n.stopChan:
			return
		case evt := <-eventChan:
			if err := n.handleEvent(evt); err != nil {
				n.logger.Error("处理事件失败",
					zap.Error(err),
					zap.Any("event", evt),
				)
			}
		}
	}
}

// handleEvent 处理单个事件
func (n *DingTalkNotifier) handleEvent(evt types.Event) error {
	switch evt.Type {
	case types.TypeLogin:
		return n.SendLoginNotification(
			evt.Username,
			fmt.Sprintf("%s:%s", evt.IP, evt.Port),
			evt.Timestamp,
			evt.ServerInfo,
		)
	case types.TypeLogout:
		return n.SendLogoutNotification(
			evt.Username,
			fmt.Sprintf("%s:%s", evt.IP, evt.Port),
			evt.Timestamp,
			evt.ServerInfo,
		)
	default:
		return fmt.Errorf("未知的事件类型: %v", evt.Type)
	}
}

// SendLoginNotification 发送登录通知
func (n *DingTalkNotifier) SendLoginNotification(username, ip string, loginTime time.Time, serverInfo *types.ServerInfo) error {
	msg := types.NotifyMessage{
		MsgType: "markdown",
		Content: map[string]interface{}{
			"title": "用户登录通知",
			"text": fmt.Sprintf("### 用户登录通知\n"+
				"**服务器：** %s\n\n"+
				"**服务器IP：** %s\n\n"+
				"**用户名：** %s\n\n"+
				"**IP地址：** %s\n\n"+
				"**登录时间：** %s",
				serverInfo.Hostname,
				serverInfo.IP,
				username,
				ip,
				loginTime.Format("2006-01-02 15:04:05")),
		},
	}
	return n.sendMessage(msg)
}

// SendLogoutNotification 发送登出通知
func (n *DingTalkNotifier) SendLogoutNotification(username, ip string, logoutTime time.Time, serverInfo *types.ServerInfo) error {
	msg := types.NotifyMessage{
		MsgType: "markdown",
		Content: map[string]interface{}{
			"title": "用户登出通知",
			"text": fmt.Sprintf("### 用户登出通知\n"+
				"**服务器：** %s\n\n"+
				"**服务器IP：** %s\n\n"+
				"**用户名：** %s\n\n"+
				"**IP地址：** %s\n\n"+
				"**登出时间：** %s",
				serverInfo.Hostname,
				serverInfo.IP,
				username,
				ip,
				logoutTime.Format("2006-01-02 15:04:05")),
		},
	}
	return n.sendMessage(msg)
}

// generateSignedURL 生成带签名的 URL
func (n *DingTalkNotifier) generateSignedURL() (string, error) {
	if n.secret == "" {
		return n.webhookURL, nil
	}

	timestamp := time.Now().UnixMilli()
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, n.secret)

	// 计算签名
	mac := hmac.New(sha256.New, []byte(n.secret))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// 构建带签名的 URL
	baseURL, err := url.Parse(n.webhookURL)
	if err != nil {
		return "", fmt.Errorf("解析 webhook URL 失败: %v", err)
	}

	query := baseURL.Query()
	query.Set("timestamp", fmt.Sprintf("%d", timestamp))
	query.Set("sign", signature)
	baseURL.RawQuery = query.Encode()

	return baseURL.String(), nil
}

// sendMessage 发送消息到钉钉
func (n *DingTalkNotifier) sendMessage(msg types.NotifyMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %v", err)
	}

	// 生成带签名的 URL
	signedURL, err := n.generateSignedURL()
	if err != nil {
		return fmt.Errorf("生成签名 URL 失败: %v", err)
	}

	resp, err := http.Post(signedURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("send message failed: %v", err)
	}

	// 确保响应体被关闭
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			n.logger.Error("关闭响应体失败", zap.Error(closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("send message failed with status code: %d", resp.StatusCode)
	}

	return nil
}
