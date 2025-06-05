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
								zap.String("notifier", fmt.Sprintf("%T", n)),
								zap.Error(err),
							)
						}
					case types.TypeLogout:
						err = n.SendLogoutNotification(e.Username, e.IP, e.Timestamp, e.ServerInfo)
						if err != nil {
							s.logger.Error("发送登出通知失败",
								zap.String("notifier", fmt.Sprintf("%T", n)),
								zap.Error(err),
							)
						}
					}
				}(notifier, evt)
			}
		}
	}
}

// InitNotifiers 初始化通知器
func (s *NotifyManager) InitNotifiers() error {
	// 获取飞书配置
	if viper.IsSet("notify.feishu.webhook_url") {
		webhookURL := viper.GetString("notify.feishu.webhook_url")
		if webhookURL != "" {
			feishuNotifier := NewFeishuNotifier(webhookURL, s.logger)
			s.notifiers = append(s.notifiers, feishuNotifier)
			s.logger.Info("已初始化飞书通知器")

			// 发送测试消息
			if err := feishuNotifier.sendTestMessage(); err != nil {
				s.logger.Error("飞书通知器测试失败", zap.Error(err))
			}
		}
	}

	// 获取钉钉配置
	if viper.IsSet("notify.dingtalk.webhook_url") {
		webhookURL := viper.GetString("notify.dingtalk.webhook_url")
		secret := viper.GetString("notify.dingtalk.secret")
		if webhookURL != "" {
			dingtalkNotifier := NewDingTalkNotifier(webhookURL, secret, s.logger)
			s.notifiers = append(s.notifiers, dingtalkNotifier)
			s.logger.Info("已初始化钉钉通知器")

			// 发送测试消息
			if err := dingtalkNotifier.sendTestMessage(); err != nil {
				s.logger.Error("钉钉通知器测试失败", zap.Error(err))
			}
		}
	}

	if len(s.notifiers) == 0 {
		return fmt.Errorf("没有配置任何通知器")
	}

	return nil
}
