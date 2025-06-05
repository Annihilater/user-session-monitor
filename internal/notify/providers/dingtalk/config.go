package dingtalk

import (
	"github.com/Annihilater/user-session-monitor/internal/notify/config"
)

// Config 钉钉通知器配置
type Config struct {
	WebhookURL string `json:"webhook_url" yaml:"webhook_url"`
	Secret     string `json:"secret" yaml:"secret"`
	Timeout    int    `json:"timeout" yaml:"timeout"`
	Enabled    bool   `json:"enabled" yaml:"enabled"`
}

// Validate 验证配置
func (c *Config) Validate() error {
	validator := &config.DingTalkConfigValidator{
		Options: map[string]string{
			"webhook_url": c.WebhookURL,
			"secret":      c.Secret,
		},
	}
	return validator.Validate()
}

// ToMap 将配置转换为map
func (c *Config) ToMap() map[string]string {
	return map[string]string{
		"webhook_url": c.WebhookURL,
		"secret":      c.Secret,
	}
}
