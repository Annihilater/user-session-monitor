package factory

import (
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/notify/config"
	"github.com/Annihilater/user-session-monitor/internal/notify/notifier"
)

// Factory 通知器工厂
type Factory struct {
	provider *Provider
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewFactory 创建新的工厂实例
func NewFactory(logger *zap.Logger) *Factory {
	return &Factory{
		provider: NewProvider(),
		logger:   logger,
	}
}

// Create 创建通知器实例
func (f *Factory) Create(cfg *config.Config) (notifier.Notifier, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	creator, exists := f.provider.Get(cfg.Type)
	if !exists {
		return nil, fmt.Errorf("未知的通知器类型: %s", cfg.Type)
	}

	// 验证配置
	validator := config.GetValidator(cfg.Type, cfg.Options)
	if validator != nil {
		if err := validator.Validate(); err != nil {
			return nil, fmt.Errorf("配置验证失败: %v", err)
		}
	}

	return creator(cfg, f.logger)
}
