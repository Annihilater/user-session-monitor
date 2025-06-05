package feishu

import (
	"github.com/Annihilater/user-session-monitor/internal/notify/config"
)

// Config 飞书通知器配置
type Config struct {
	WebhookURL string `json:"webhook_url" yaml:"webhook_url"`
	Timeout    int    `json:"timeout" yaml:"timeout"`
	Enabled    bool   `json:"enabled" yaml:"enabled"`
}

// Validate 验证配置
func (c *Config) Validate() error {
	validator := &config.FeishuConfigValidator{
		Options: map[string]string{
			"webhook_url": c.WebhookURL,
		},
	}
	return validator.Validate()
}

// ToMap 将配置转换为map
func (c *Config) ToMap() map[string]string {
	return map[string]string{
		"webhook_url": c.WebhookURL,
	}
}
