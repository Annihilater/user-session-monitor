package config

import "fmt"

// Validator 配置验证器接口
type Validator interface {
	Validate() error
}

// RequiredOption 必需的配置选项
type RequiredOption struct {
	Name        string
	Description string
}

// ValidateRequiredOptions 验证必需的配置选项
func ValidateRequiredOptions(options map[string]string, required []RequiredOption) error {
	for _, opt := range required {
		if value, exists := options[opt.Name]; !exists || value == "" {
			return fmt.Errorf("缺少必需的配置项 %s: %s", opt.Name, opt.Description)
		}
	}
	return nil
}

// EmailConfigValidator 邮件配置验证器
type EmailConfigValidator struct {
	Options map[string]string
}

func (v *EmailConfigValidator) Validate() error {
	required := []RequiredOption{
		{Name: "host", Description: "SMTP 服务器地址"},
		{Name: "port", Description: "SMTP 服务器端口"},
		{Name: "username", Description: "SMTP 用户名"},
		{Name: "password", Description: "SMTP 密码"},
		{Name: "from", Description: "发件人地址"},
		{Name: "to", Description: "收件人地址"},
	}
	return ValidateRequiredOptions(v.Options, required)
}

// DingTalkConfigValidator 钉钉配置验证器
type DingTalkConfigValidator struct {
	Options map[string]string
}

func (v *DingTalkConfigValidator) Validate() error {
	required := []RequiredOption{
		{Name: "webhook_url", Description: "Webhook URL"},
	}
	return ValidateRequiredOptions(v.Options, required)
}

// FeishuConfigValidator 飞书配置验证器
type FeishuConfigValidator struct {
	Options map[string]string
}

func (v *FeishuConfigValidator) Validate() error {
	required := []RequiredOption{
		{Name: "webhook_url", Description: "Webhook URL"},
	}
	return ValidateRequiredOptions(v.Options, required)
}

// TelegramConfigValidator Telegram配置验证器
type TelegramConfigValidator struct {
	Options map[string]string
}

func (v *TelegramConfigValidator) Validate() error {
	required := []RequiredOption{
		{Name: "bot_token", Description: "Bot Token"},
		{Name: "chat_id", Description: "Chat ID"},
	}
	return ValidateRequiredOptions(v.Options, required)
}

// GetValidator 获取配置验证器
func GetValidator(typ NotifierType, options map[string]string) Validator {
	switch typ {
	case TypeEmail:
		return &EmailConfigValidator{Options: options}
	case TypeDingTalk:
		return &DingTalkConfigValidator{Options: options}
	case TypeFeishu:
		return &FeishuConfigValidator{Options: options}
	case TypeTelegram:
		return &TelegramConfigValidator{Options: options}
	default:
		return nil
	}
}
