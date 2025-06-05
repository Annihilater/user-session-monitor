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
			Enabled: true, // Telegram 默认不使用 enabled 字段
			Config: map[string]string{
				"bot_token": viper.GetString("notify.telegram.bot_token"),
				"chat_id":   viper.GetString("notify.telegram.chat_id"),
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
	}

	// 验证是否至少有一个通知器被初始化
	if len(s.notifiers) == 0 {
		return fmt.Errorf("没有配置任何通知器")
	}

	// 测试所有通知器
	for _, notifier := range s.notifiers {
		if err := notifier.sendTestMessage(); err != nil {
			s.logger.Error("通知器测试失败",
				zap.String("notifier_zh", notifier.GetNameZh()),
				zap.String("notifier_en", notifier.GetNameEn()),
				zap.Error(err),
			)
			return fmt.Errorf("通知器测试失败: %v", err)
		}
	}

	return nil
}
