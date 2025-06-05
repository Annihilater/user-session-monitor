package notify

import (
	"fmt"
	"sync"
	"time"

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
			// 验证 webhook URL
			if err := notifier.sendTestMessage(); err != nil {
				s.logger.Error("飞书 webhook URL 验证失败",
					zap.String("url", webhookURL),
					zap.Error(err),
				)
				return fmt.Errorf("飞书 webhook URL 验证失败: %v", err)
			}
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
			// 验证 webhook URL
			if err := notifier.sendTestMessage(); err != nil {
				s.logger.Error("钉钉 webhook URL 验证失败",
					zap.String("url", webhookURL),
					zap.Error(err),
				)
				return fmt.Errorf("钉钉 webhook URL 验证失败: %v", err)
			}
			s.notifiers = append(s.notifiers, notifier)
		}
	}

	if len(s.notifiers) == 0 {
		s.logger.Warn("没有配置任何通知器")
	}

	return nil
}

// Start 启动通知服务
func (s *NotifyManager) Start(eventChan <-chan types.Event) {
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
			// 并发发送通知，带重试机制
			var wg sync.WaitGroup
			for _, notifier := range s.notifiers {
				wg.Add(1)
				go func(n Notifier) {
					defer wg.Done()
					// 重试3次
					for i := 0; i < 3; i++ {
						if err := s.handleEvent(n, evt); err != nil {
							s.logger.Error("发送通知失败，准备重试",
								zap.Error(err),
								zap.Int("retry", i+1),
								zap.Any("event", evt),
							)
							time.Sleep(time.Second * time.Duration(i+1))
							continue
						}
						return
					}
					s.logger.Error("发送通知最终失败",
						zap.Any("event", evt),
					)
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
