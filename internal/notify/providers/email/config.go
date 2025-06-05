package email

import (
	"strings"

	"github.com/Annihilater/user-session-monitor/internal/notify/config"
)

// Config 邮件通知器配置
type Config struct {
	Host     string `json:"host" yaml:"host"`
	Port     string `json:"port" yaml:"port"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	From     string `json:"from" yaml:"from"`
	To       string `json:"to" yaml:"to"`
	Timeout  int    `json:"timeout" yaml:"timeout"`
	Enabled  bool   `json:"enabled" yaml:"enabled"`
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 如果未配置发件人，使用用户名
	if c.From == "" {
		c.From = c.Username
	}

	validator := &config.EmailConfigValidator{
		Options: map[string]string{
			"host":     c.Host,
			"port":     c.Port,
			"username": c.Username,
			"password": c.Password,
			"from":     c.From,
			"to":       c.To,
		},
	}
	return validator.Validate()
}

// ToMap 将配置转换为map
func (c *Config) ToMap() map[string]string {
	return map[string]string{
		"host":     c.Host,
		"port":     c.Port,
		"username": c.Username,
		"password": c.Password,
		"from":     c.From,
		"to":       c.To,
	}
}

// GetRecipients 获取收件人列表
func (c *Config) GetRecipients() []string {
	if c.To == "" {
		return nil
	}
	return strings.Split(c.To, ",")
}
