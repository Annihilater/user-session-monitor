package feishu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Notifier struct {
	webhookURL string
}

type Message struct {
	MsgType string                 `json:"msg_type"`
	Content map[string]interface{} `json:"content"`
}

func NewNotifier(webhookURL string) *Notifier {
	return &Notifier{
		webhookURL: webhookURL,
	}
}

func (n *Notifier) SendLoginNotification(username, ip string, loginTime time.Time) error {
	msg := Message{
		MsgType: "text",
		Content: map[string]interface{}{
			"text": fmt.Sprintf("用户登录通知\n用户名：%s\nIP地址：%s\n登录时间：%s",
				username, ip, loginTime.Format("2006-01-02 15:04:05")),
		},
	}
	return n.sendMessage(msg)
}

func (n *Notifier) SendLogoutNotification(username string, logoutTime time.Time) error {
	msg := Message{
		MsgType: "text",
		Content: map[string]interface{}{
			"text": fmt.Sprintf("用户登出通知\n用户名：%s\n登出时间：%s",
				username, logoutTime.Format("2006-01-02 15:04:05")),
		},
	}
	return n.sendMessage(msg)
}

func (n *Notifier) sendMessage(msg Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %v", err)
	}

	resp, err := http.Post(n.webhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("send message failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("send message failed with status code: %d", resp.StatusCode)
	}

	return nil
}
