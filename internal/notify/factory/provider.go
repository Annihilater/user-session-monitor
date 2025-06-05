package factory

import (
	"sync"

	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/notify/config"
	"github.com/Annihilater/user-session-monitor/internal/notify/notifier"
	"github.com/Annihilater/user-session-monitor/internal/notify/providers/dingtalk"
	"github.com/Annihilater/user-session-monitor/internal/notify/providers/email"
	"github.com/Annihilater/user-session-monitor/internal/notify/providers/feishu"
	"github.com/Annihilater/user-session-monitor/internal/notify/providers/telegram"
)

// Creator 定义通知器创建函数类型
type Creator func(cfg *config.Config, logger *zap.Logger) (notifier.Notifier, error)

// Provider 通知器提供者
type Provider struct {
	creators map[config.NotifierType]Creator
	mu       sync.RWMutex
}

// NewProvider 创建新的提供者
func NewProvider() *Provider {
	p := &Provider{
		creators: make(map[config.NotifierType]Creator),
	}
	p.registerDefaultProviders()
	return p
}

// Register 注册通知器创建函数
func (p *Provider) Register(typ config.NotifierType, creator Creator) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.creators[typ] = creator
}

// Get 获取通知器创建函数
func (p *Provider) Get(typ config.NotifierType) (Creator, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	creator, exists := p.creators[typ]
	return creator, exists
}

// registerDefaultProviders 注册默认的通知器提供者
func (p *Provider) registerDefaultProviders() {
	// 注册邮件通知器
	p.Register(config.TypeEmail, func(cfg *config.Config, logger *zap.Logger) (notifier.Notifier, error) {
		return email.NewEmailNotifier(cfg, logger)
	})

	// 注册飞书通知器
	p.Register(config.TypeFeishu, func(cfg *config.Config, logger *zap.Logger) (notifier.Notifier, error) {
		return feishu.NewFeishuNotifier(cfg, logger)
	})

	// 注册钉钉通知器
	p.Register(config.TypeDingTalk, func(cfg *config.Config, logger *zap.Logger) (notifier.Notifier, error) {
		return dingtalk.NewDingTalkNotifier(cfg, logger)
	})

	// 注册 Telegram 通知器
	p.Register(config.TypeTelegram, func(cfg *config.Config, logger *zap.Logger) (notifier.Notifier, error) {
		return telegram.NewTelegramNotifier(cfg, logger)
	})
}
