package notify

import (
	"go.uber.org/zap"
)

// Factory 通知器工厂
type Factory struct {
	logger *zap.Logger
}

// NewFactory 创建新的通知器工厂
func NewFactory(logger *zap.Logger) *Factory {
	return &Factory{
		logger: logger,
	}
}

// CreateNotifier 根据配置创建通知器
func (f *Factory) CreateNotifier(config map[string]interface{}) []interface{} {
	var notifiers []interface{}

	// 检查飞书通知器配置
	if feishuConfig, ok := config["feishu"].(map[string]interface{}); ok {
		if enabled, _ := feishuConfig["enabled"].(bool); enabled {
			if webhookURL, ok := feishuConfig["webhook_url"].(string); ok && webhookURL != "" {
				notifier := NewFeishuNotifier(webhookURL, f.logger)
				notifiers = append(notifiers, notifier)
			}
		}
	}

	// 检查钉钉通知器配置
	if dingtalkConfig, ok := config["dingtalk"].(map[string]interface{}); ok {
		if enabled, _ := dingtalkConfig["enabled"].(bool); enabled {
			if webhookURL, ok := dingtalkConfig["webhook_url"].(string); ok && webhookURL != "" {
				secret, _ := dingtalkConfig["secret"].(string)
				notifier := NewDingTalkNotifier(webhookURL, secret, f.logger)
				notifiers = append(notifiers, notifier)
			}
		}
	}

	return notifiers
}
