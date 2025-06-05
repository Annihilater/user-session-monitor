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
	// 检查飞书通知器配置
	if viper.GetBool("notify.feishu.enabled") {
		webhookURL := viper.GetString("notify.feishu.webhook_url")
		if webhookURL != "" {
			s.logger.Info("初始化飞书通知器")
			notifier := NewFeishuNotifier(webhookURL, s.logger)
			s.notifiers = append(s.notifiers, notifier)
		}
	}

	// 检查钉钉通知器配置
	if viper.GetBool("notify.dingtalk.enabled") {
		webhookURL := viper.GetString("notify.dingtalk.webhook_url")
		if webhookURL != "" {
			s.logger.Info("初始化钉钉通知器")
			secret := viper.GetString("notify.dingtalk.secret")
			notifier := NewDingTalkNotifier(webhookURL, secret, s.logger)
			s.notifiers = append(s.notifiers, notifier)
		}
	}

	return nil
}

// Start 启动通知服务
func (s *NotifyManager) Start(eventChan <-chan types.Event) {
	s.wg.Add(1)
	go s.processEvents(eventChan)

	// 启动所有通知器
	for _, notifier := range s.notifiers {
		notifier.Start(eventChan)
	}
}

// Stop 停止通知服务
func (s *NotifyManager) Stop() {
	close(s.stopChan)
	// 停止所有通知器
	for _, notifier := range s.notifiers {
		notifier.Stop()
	}
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
			// 并发发送通知
			var wg sync.WaitGroup
			for _, notifier := range s.notifiers {
				wg.Add(1)
				go func(n Notifier) {
					defer wg.Done()
					if err := s.handleEvent(n, evt); err != nil {
						s.logger.Error("处理事件失败",
							zap.Error(err),
							zap.Any("event", evt),
						)
					}
				}(notifier)
			}
			wg.Wait()
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
