package notify

import (
	"fmt"
	"sync"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/event"
	"github.com/Annihilater/user-session-monitor/internal/notify/config"
	"github.com/Annihilater/user-session-monitor/internal/notify/factory"
	"github.com/Annihilater/user-session-monitor/internal/notify/notifier"
	"github.com/Annihilater/user-session-monitor/internal/types"
)

// NotifyManager 通知管理器
type NotifyManager struct {
	notifiers []notifier.Notifier
	logger    *zap.Logger
	factory   *factory.Factory
	mu        sync.RWMutex
}

// NewNotifyManager 创建新的通知管理器
func NewNotifyManager(logger *zap.Logger) *NotifyManager {
	return &NotifyManager{
		notifiers: make([]notifier.Notifier, 0),
		logger:    logger,
		factory:   factory.NewFactory(logger),
	}
}

// InitNotifiers 初始化所有通知器
func (m *NotifyManager) InitNotifiers() error {
	// 获取所有启用的通知器配置
	notifierConfigs := m.getEnabledNotifierConfigs()

	// 初始化每个通知器
	for _, cfg := range notifierConfigs {
		n, err := m.factory.Create(cfg)
		if err != nil {
			m.logger.Warn("创建通知器失败",
				zap.String("type", string(cfg.Type)),
				zap.Error(err),
			)
			continue
		}

		// 初始化通知器
		if err := n.Initialize(); err != nil {
			m.logger.Warn("初始化通知器失败",
				zap.String("type", string(cfg.Type)),
				zap.Error(err),
			)
			continue
		}

		// 添加到通知器列表
		m.mu.Lock()
		m.notifiers = append(m.notifiers, n)
		m.mu.Unlock()
	}

	// 检查是否有可用的通知器
	if len(m.notifiers) == 0 {
		return fmt.Errorf("没有可用的通知器")
	}

	return nil
}

// Start 启动通知管理器
func (m *NotifyManager) Start(eventBus *event.Bus) {
	// 订阅事件
	eventChan := eventBus.Subscribe()
	go func() {
		for e := range eventChan {
			switch e.Type {
			case types.TypeLogin:
				m.handleLoginEvent(e)
			case types.TypeLogout:
				m.handleLogoutEvent(e)
			}
		}
	}()
}

// Stop 停止通知管理器
func (m *NotifyManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifiers = nil
}

// handleLoginEvent 处理登录事件
func (m *NotifyManager) handleLoginEvent(e types.Event) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, n := range m.notifiers {
		if !n.IsEnabled() {
			continue
		}

		go func(notifier notifier.Notifier) {
			if err := notifier.SendLoginNotification(e.Username, e.IP, e.Timestamp, e.ServerInfo); err != nil {
				nameZh, nameEn := notifier.GetName()
				m.logger.Error("发送登录通知失败",
					zap.String("notifier_zh", nameZh),
					zap.String("notifier_en", nameEn),
					zap.Error(err),
				)
			}
		}(n)
	}
}

// handleLogoutEvent 处理登出事件
func (m *NotifyManager) handleLogoutEvent(e types.Event) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, n := range m.notifiers {
		if !n.IsEnabled() {
			continue
		}

		go func(notifier notifier.Notifier) {
			if err := notifier.SendLogoutNotification(e.Username, e.IP, e.Timestamp, e.ServerInfo); err != nil {
				nameZh, nameEn := notifier.GetName()
				m.logger.Error("发送登出通知失败",
					zap.String("notifier_zh", nameZh),
					zap.String("notifier_en", nameEn),
					zap.Error(err),
				)
			}
		}(n)
	}
}

// getEnabledNotifierConfigs 获取所有启用的通知器配置
func (m *NotifyManager) getEnabledNotifierConfigs() []*config.Config {
	var configs []*config.Config

	// 检查每种通知器类型
	notifierTypes := []config.NotifierType{
		config.TypeEmail,
		config.TypeFeishu,
		config.TypeDingTalk,
		config.TypeTelegram,
	}

	for _, typ := range notifierTypes {
		// 检查是否启用
		enabled := viper.GetBool(fmt.Sprintf("notify.%s.enabled", typ))
		if !enabled {
			continue
		}

		// 创建配置
		cfg := config.NewConfig(typ)

		// 获取超时设置
		timeoutSeconds := viper.GetFloat64(fmt.Sprintf("notify.%s.timeout", typ))
		if timeoutSeconds > 0 {
			cfg.Timeout = config.GetTimeout(timeoutSeconds)
		}

		// 获取所有配置选项
		options := viper.GetStringMapString(fmt.Sprintf("notify.%s", typ))
		for k, v := range options {
			if k != "enabled" && k != "timeout" {
				cfg.Options[k] = v
			}
		}

		configs = append(configs, cfg)
	}

	return configs
}
