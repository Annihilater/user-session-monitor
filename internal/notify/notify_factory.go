package notify

import (
	"fmt"

	"go.uber.org/zap"
)

// RegisterNotifier 注册通知器工厂函数
func RegisterNotifier(notifierType NotifierType, factory NotifierFactory) {
	notifierFactories[notifierType] = factory
}

// CreateNotifier 创建通知器实例
func CreateNotifier(config NotifierConfig, logger *zap.Logger) (Notifier, error) {
	factory, exists := notifierFactories[config.Type]
	if !exists {
		return nil, fmt.Errorf("未知的通知器类型: %s", config.Type)
	}
	return factory(config, logger)
}

func init() {
	// 注册飞书通知器
	RegisterNotifier(NotifierTypeFeishu, func(config NotifierConfig, logger *zap.Logger) (Notifier, error) {
		webhookURL, exists := config.Config["webhook_url"]
		if !exists || webhookURL == "" {
			return nil, fmt.Errorf("飞书通知器缺少 webhook_url 配置")
		}
		return NewFeishuNotifier(webhookURL, logger), nil
	})

	// 注册钉钉通知器
	RegisterNotifier(NotifierTypeDingTalk, func(config NotifierConfig, logger *zap.Logger) (Notifier, error) {
		webhookURL, exists := config.Config["webhook_url"]
		if !exists || webhookURL == "" {
			return nil, fmt.Errorf("钉钉通知器缺少 webhook_url 配置")
		}
		secret := config.Config["secret"] // secret 可选
		return NewDingTalkNotifier(webhookURL, secret, logger), nil
	})

	// 注册 Telegram 通知器
	RegisterNotifier(NotifierTypeTelegram, func(config NotifierConfig, logger *zap.Logger) (Notifier, error) {
		botToken, exists := config.Config["bot_token"]
		if !exists || botToken == "" {
			return nil, fmt.Errorf("telegram 通知器缺少 bot_token 配置")
		}
		chatID, exists := config.Config["chat_id"]
		if !exists || chatID == "" {
			return nil, fmt.Errorf("telegram 通知器缺少 chat_id 配置")
		}
		return NewTelegramNotifier(botToken, chatID, logger), nil
	})
}
