package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/types"
)

// FeishuNotifier 飞书通知器
type FeishuNotifier struct {
	webhookURL string
	logger     *zap.Logger
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// NewFeishuNotifier 创建新的飞书通知器
func NewFeishuNotifier(webhookURL string, logger *zap.Logger) *FeishuNotifier {
	return &FeishuNotifier{
		webhookURL: webhookURL,
		logger:     logger,
		stopChan:   make(chan struct{}),
	}
}

// Start 启动通知处理器
func (n *FeishuNotifier) Start(eventChan <-chan types.Event) {
	n.wg.Add(1)
	go n.processEvents(eventChan)
}

// Stop 停止通知处理器
func (n *FeishuNotifier) Stop() {
	close(n.stopChan)
	n.wg.Wait()
}

// processEvents 处理事件
func (n *FeishuNotifier) processEvents(eventChan <-chan types.Event) {
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
func (n *FeishuNotifier) handleEvent(evt types.Event) error {
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

func (n *FeishuNotifier) SendLoginNotification(username, ip string, loginTime time.Time, serverInfo *types.ServerInfo) error {
	msg := types.NotifyMessage{
		MsgType: "text",
		Content: map[string]interface{}{
			"text": fmt.Sprintf("用户登录通知\n服务器：%s\n服务器IP：%s\n用户名：%s\nIP地址：%s\n登录时间：%s",
				serverInfo.Hostname,
				serverInfo.IP,
				username,
				ip,
				loginTime.Format("2006-01-02 15:04:05")),
		},
	}
	return n.sendMessage(msg)
}

func (n *FeishuNotifier) SendLogoutNotification(username, ip string, logoutTime time.Time, serverInfo *types.ServerInfo) error {
	msg := types.NotifyMessage{
		MsgType: "text",
		Content: map[string]interface{}{
			"text": fmt.Sprintf("用户登出通知\n服务器：%s\n服务器IP：%s\n用户名：%s\nIP地址：%s\n登出时间：%s",
				serverInfo.Hostname,
				serverInfo.IP,
				username,
				ip,
				logoutTime.Format("2006-01-02 15:04:05")),
		},
	}
	return n.sendMessage(msg)
}

func (n *FeishuNotifier) sendMessage(msg types.NotifyMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %v", err)
	}

	resp, err := http.Post(n.webhookURL, "application/json", bytes.NewBuffer(payload))
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
