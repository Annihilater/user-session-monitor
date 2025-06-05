package telegram

import (
	"github.com/Annihilater/user-session-monitor/internal/notify/config"
)

// Config Telegram通知器配置
type Config struct {
	BotToken string `json:"bot_token" yaml:"bot_token"`
	ChatID   string `json:"chat_id" yaml:"chat_id"`
	Timeout  int    `json:"timeout" yaml:"timeout"`
	Enabled  bool   `json:"enabled" yaml:"enabled"`
}

// Validate 验证配置
func (c *Config) Validate() error {
	validator := &config.TelegramConfigValidator{
		Options: map[string]string{
			"bot_token": c.BotToken,
			"chat_id":   c.ChatID,
		},
	}
	return validator.Validate()
}

// ToMap 将配置转换为map
func (c *Config) ToMap() map[string]string {
	return map[string]string{
		"bot_token": c.BotToken,
		"chat_id":   c.ChatID,
	}
}
