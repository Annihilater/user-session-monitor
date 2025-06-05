package notifier

import (
	"time"

	"github.com/Annihilater/user-session-monitor/internal/types"
)

// Notifier 定义通知器接口
type Notifier interface {
	// SendLoginNotification 发送登录通知
	SendLoginNotification(username, ip string, loginTime time.Time, serverInfo *types.ServerInfo) error

	// SendLogoutNotification 发送登出通知
	SendLogoutNotification(username, ip string, logoutTime time.Time, serverInfo *types.ServerInfo) error

	// Initialize 初始化通知器
	Initialize() error

	// IsEnabled 检查通知器是否启用
	IsEnabled() bool

	// GetName 获取通知器名称
	GetName() (string, string) // 返回 (中文名, 英文名)
}
