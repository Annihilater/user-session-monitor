package notify

import (
	"fmt"
	"sync"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/event"
	"github.com/Annihilater/user-session-monitor/internal/types"
)

// NotifyManager 通知服务管理器
type NotifyManager struct {
	notifiers []Notifier
	logger    *zap.Logger
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// NewNotifyManager 创建新的通知服务管理器
func NewNotifyManager(logger *zap.Logger) *NotifyManager {
	return &NotifyManager{
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

// Start 启动通知服务
func (s *NotifyManager) Start(eventBus *event.Bus) {
	eventChan := eventBus.Subscribe()

	// 启动事件处理协程
	s.wg.Add(1)
	go s.processEvents(eventChan)
}

// Stop 停止通知服务
func (s *NotifyManager) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

// processEvents 处理事件
func (s *NotifyManager) processEvents(eventChan <-chan types.Event) {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopChan:
			return
		case evt := <-eventChan:
			// 为每个通知器并发处理事件
			for _, notifier := range s.notifiers {
				go func(n Notifier, e types.Event) {
					var err error
					switch e.Type {
					case types.TypeLogin:
						err = n.SendLoginNotification(e.Username, e.IP, e.Timestamp, e.ServerInfo)
						if err != nil {
							s.logger.Error("发送登录通知失败",
								zap.String("notifier_zh", n.GetNameZh()),
								zap.String("notifier_en", n.GetNameEn()),
								zap.Error(err),
							)
						}
					case types.TypeLogout:
						err = n.SendLogoutNotification(e.Username, e.IP, e.Timestamp, e.ServerInfo)
						if err != nil {
							s.logger.Error("发送登出通知失败",
								zap.String("notifier_zh", n.GetNameZh()),
								zap.String("notifier_en", n.GetNameEn()),
								zap.Error(err),
							)
						}
					}
				}(notifier, evt)
			}
		}
	}
}

// InitNotifiers 初始化所有通知器
func (s *NotifyManager) InitNotifiers() error {
	// 预定义的通知器配置
	notifierConfigs := []NotifierConfig{
		{
			Type:    NotifierTypeFeishu,
			NameZh:  "飞书",
			NameEn:  "Feishu",
			Enabled: viper.GetBool("notify.feishu.enabled"),
			Config: map[string]string{
				"webhook_url": viper.GetString("notify.feishu.webhook_url"),
			},
		},
		{
			Type:    NotifierTypeDingTalk,
			NameZh:  "钉钉",
			NameEn:  "DingTalk",
			Enabled: viper.GetBool("notify.dingtalk.enabled"),
			Config: map[string]string{
				"webhook_url": viper.GetString("notify.dingtalk.webhook_url"),
				"secret":      viper.GetString("notify.dingtalk.secret"),
			},
		},
		{
			Type:    NotifierTypeTelegram,
			NameZh:  "电报",
			NameEn:  "Telegram",
			Enabled: viper.GetBool("notify.telegram.enabled"),
			Config: map[string]string{
				"bot_token": viper.GetString("notify.telegram.bot_token"),
				"chat_id":   viper.GetString("notify.telegram.chat_id"),
			},
		},
		{
			Type:    NotifierTypeEmail,
			NameZh:  "邮件",
			NameEn:  "Email",
			Enabled: viper.GetBool("notify.email.enabled"),
			Config: map[string]string{
				"host":     viper.GetString("notify.email.host"),
				"port":     viper.GetString("notify.email.port"),
				"username": viper.GetString("notify.email.username"),
				"password": viper.GetString("notify.email.password"),
				"from":     viper.GetString("notify.email.from"),
				"to":       viper.GetString("notify.email.to"),
			},
		},
	}

	// 遍历配置创建通知器
	for _, config := range notifierConfigs {
		if !config.Enabled {
			s.logger.Info("通知器未启用，跳过",
				zap.String("type", string(config.Type)),
				zap.String("name_zh", config.NameZh),
				zap.String("name_en", config.NameEn),
			)
			continue
		}

		notifier, err := CreateNotifier(config, s.logger)
		if err != nil {
			s.logger.Error("创建通知器失败",
				zap.String("type", string(config.Type)),
				zap.String("name_zh", config.NameZh),
				zap.String("name_en", config.NameEn),
				zap.Error(err),
			)
			continue
		}

		s.logger.Info("初始化通知器",
			zap.String("type", string(config.Type)),
			zap.String("name_zh", config.NameZh),
			zap.String("name_en", config.NameEn),
		)
		s.notifiers = append(s.notifiers, notifier)

		// 在后台发送测试消息
		go func(n Notifier) {
			if err := n.sendTestMessage(); err != nil {
				s.logger.Warn("通知器测试失败",
					zap.String("notifier_zh", n.GetNameZh()),
					zap.String("notifier_en", n.GetNameEn()),
					zap.Error(err),
				)
			}
		}(notifier)
	}

	// 验证是否至少有一个通知器被初始化
	if len(s.notifiers) == 0 {
		return fmt.Errorf("没有配置任何通知器")
	}

	return nil
}
