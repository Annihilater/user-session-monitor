package notify

import (
	"time"

	"github.com/Annihilater/user-session-monitor/internal/types"
)

// Notifier 通知器接口
type Notifier interface {
	// SendLoginNotification 发送登录通知
	SendLoginNotification(username, ip string, loginTime time.Time, serverInfo *types.ServerInfo) error

	// SendLogoutNotification 发送登出通知
	SendLogoutNotification(username, ip string, logoutTime time.Time, serverInfo *types.ServerInfo) error

	// sendTestMessage 发送测试消息
	sendTestMessage() error

	// GetNameZh 获取通知器中文名
	GetNameZh() string

	// GetNameEn 获取通知器英文名
	GetNameEn() string
}

// BaseNotifier 基础通知器
type BaseNotifier struct {
	stopChan chan struct{}
	nameZh   string // 通知器中文名
	nameEn   string // 通知器英文名
}

// NewBaseNotifier 创建基础通知器
func NewBaseNotifier(nameZh, nameEn string) *BaseNotifier {
	return &BaseNotifier{
		stopChan: make(chan struct{}),
		nameZh:   nameZh,
		nameEn:   nameEn,
	}
}

// GetNameZh 获取通知器中文名
func (n *BaseNotifier) GetNameZh() string {
	return n.nameZh
}

// GetNameEn 获取通知器英文名
func (n *BaseNotifier) GetNameEn() string {
	return n.nameEn
}

// Stop 停止通知器
func (n *BaseNotifier) Stop() {
	close(n.stopChan)
}

// IsStopped 检查是否已停止
func (n *BaseNotifier) IsStopped() bool {
	select {
	case <-n.stopChan:
		return true
	default:
		return false
	}
}
