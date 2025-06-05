package notify

import (
	"fmt"
	"sync"

	"github.com/spf13/viper"
	"go.uber.org/zap"

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

// InitNotifiers 初始化所有已配置的通知器
func (s *NotifyManager) InitNotifiers() error {
	var initErrors []string

	// 检查飞书通知器配置
	if viper.GetBool("notify.feishu.enabled") {
		webhookURL := viper.GetString("notify.feishu.webhook_url")
		if webhookURL != "" {
			s.logger.Info("初始化飞书通知器")
			notifier := NewFeishuNotifier(webhookURL, s.logger)
			// 验证 webhook URL
			if err := notifier.sendTestMessage(); err != nil {
				s.logger.Error("飞书 webhook URL 验证失败",
					zap.String("url", webhookURL),
					zap.Error(err),
				)
				initErrors = append(initErrors, fmt.Sprintf("飞书 webhook URL 验证失败: %v", err))
			} else {
				s.notifiers = append(s.notifiers, notifier)
			}
		}
	}

	// 检查钉钉通知器配置
	if viper.GetBool("notify.dingtalk.enabled") {
		webhookURL := viper.GetString("notify.dingtalk.webhook_url")
		if webhookURL != "" {
			s.logger.Info("初始化钉钉通知器")
			secret := viper.GetString("notify.dingtalk.secret")
			notifier := NewDingTalkNotifier(webhookURL, secret, s.logger)
			// 验证 webhook URL
			if err := notifier.sendTestMessage(); err != nil {
				s.logger.Error("钉钉 webhook URL 验证失败",
					zap.String("url", webhookURL),
					zap.Error(err),
				)
				initErrors = append(initErrors, fmt.Sprintf("钉钉 webhook URL 验证失败: %v", err))
			} else {
				s.notifiers = append(s.notifiers, notifier)
			}
		}
	}

	if len(s.notifiers) == 0 {
		if len(initErrors) > 0 {
			return fmt.Errorf("所有通知器初始化失败: %v", initErrors)
		}
		s.logger.Warn("没有配置任何通知器")
	}

	return nil
}

// Start 启动通知服务
func (s *NotifyManager) Start(eventChan <-chan types.Event) {
	// 为每个通知器启动独立的处理协程
	for _, notifier := range s.notifiers {
		s.wg.Add(1)
		// 为每个通知器创建独立的事件通道
		notifierChan := make(chan types.Event, 100)
		go s.processEventsForNotifier(notifierChan, notifier)

		// 启动一个协程来转发事件
		go func(ch chan<- types.Event) {
			for evt := range eventChan {
				ch <- evt
			}
			close(ch)
		}(notifierChan)
	}
}

// Stop 停止通知服务
func (s *NotifyManager) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

// processEventsForNotifier 为单个通知器处理事件
func (s *NotifyManager) processEventsForNotifier(eventChan <-chan types.Event, notifier Notifier) {
	defer s.wg.Done()

	// 获取通知器类型名称用于日志
	notifierType := fmt.Sprintf("%T", notifier)

	for {
		select {
		case <-s.stopChan:
			s.logger.Info("通知器停止工作",
				zap.String("notifier_type", notifierType),
			)
			return
		case evt, ok := <-eventChan:
			if !ok {
				s.logger.Info("事件通道已关闭",
					zap.String("notifier_type", notifierType),
				)
				return
			}
			// 在独立的协程中处理消息发送，这样不会阻塞事件接收
			go func(e types.Event) {
				if err := s.handleEvent(notifier, e); err != nil {
					s.logger.Error("发送通知失败",
						zap.String("notifier_type", notifierType),
						zap.Error(err),
						zap.Any("event", e),
					)
				} else {
					s.logger.Info("发送通知成功",
						zap.String("notifier_type", notifierType),
						zap.Any("event_type", e.Type),
					)
				}
			}(evt)
		}
	}
}

// handleEvent 处理单个事件
func (s *NotifyManager) handleEvent(notifier Notifier, evt types.Event) error {
	switch evt.Type {
	case types.TypeLogin:
		return notifier.SendLoginNotification(
			evt.Username,
			fmt.Sprintf("%s:%s", evt.IP, evt.Port),
			evt.Timestamp,
			evt.ServerInfo,
		)
	case types.TypeLogout:
		return notifier.SendLogoutNotification(
			evt.Username,
			fmt.Sprintf("%s:%s", evt.IP, evt.Port),
			evt.Timestamp,
			evt.ServerInfo,
		)
	default:
		return fmt.Errorf("未知的事件类型: %v", evt.Type)
	}
}
