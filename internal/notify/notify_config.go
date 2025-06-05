package notify

import "go.uber.org/zap"

// NotifierType 通知器类型
type NotifierType string

const (
	NotifierTypeFeishu   NotifierType = "feishu"
	NotifierTypeDingTalk NotifierType = "dingtalk"
	NotifierTypeTelegram NotifierType = "telegram"
	NotifierTypeEmail    NotifierType = "email" // 新增邮件通知器类型
)

// NotifierConfig 通知器配置
type NotifierConfig struct {
	Type    NotifierType      `json:"type"`    // 通知器类型
	NameZh  string            `json:"name_zh"` // 中文名称
	NameEn  string            `json:"name_en"` // 英文名称
	Enabled bool              `json:"enabled"` // 是否启用
	Config  map[string]string `json:"config"`  // 具体配置项
}

// NotifierFactory 通知器工厂函数类型
type NotifierFactory func(config NotifierConfig, logger *zap.Logger) (Notifier, error)

// notifierFactories 存储所有通知器的工厂函数
var notifierFactories = map[NotifierType]NotifierFactory{}
