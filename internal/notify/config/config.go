package config

import "time"

// NotifierType 通知器类型
type NotifierType string

const (
	TypeEmail    NotifierType = "email"
	TypeFeishu   NotifierType = "feishu"
	TypeDingTalk NotifierType = "dingtalk"
	TypeTelegram NotifierType = "telegram"
)

// Config 通知器配置
type Config struct {
	Type    NotifierType      // 通知器类型
	Options map[string]string // 配置选项
	Timeout time.Duration     // 超时设置
	Enabled bool              // 是否启用
}

// NewConfig 创建新的配置
func NewConfig(notifierType NotifierType) *Config {
	return &Config{
		Type:    notifierType,
		Options: make(map[string]string),
		Timeout: 3 * time.Second, // 默认超时时间
		Enabled: true,            // 默认启用
	}
}

// GetTimeout 获取超时时间
func GetTimeout(seconds float64) time.Duration {
	if seconds <= 0 {
		seconds = 3 // 默认3秒
	}
	return time.Duration(seconds * float64(time.Second))
}
