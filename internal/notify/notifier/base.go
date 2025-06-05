package notifier

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// BaseNotifier 提供基础的通知器实现
type BaseNotifier struct {
	nameZh  string        // 中文名称
	nameEn  string        // 英文名称
	timeout time.Duration // 超时设置
	logger  *zap.Logger   // 日志器
}

// NewBaseNotifier 创建一个新的基础通知器
func NewBaseNotifier(nameZh, nameEn string, timeout time.Duration, logger *zap.Logger) *BaseNotifier {
	return &BaseNotifier{
		nameZh:  nameZh,
		nameEn:  nameEn,
		timeout: timeout,
		logger:  logger,
	}
}

// GetName 获取通知器名称
func (n *BaseNotifier) GetName() (string, string) {
	return n.nameZh, n.nameEn
}

// IsEnabled 默认实现返回 true
func (n *BaseNotifier) IsEnabled() bool {
	return true
}

// Initialize 默认实现
func (n *BaseNotifier) Initialize() error {
	return nil
}

// InitializeWithTest 提供带测试消息的初始化实现
func (n *BaseNotifier) InitializeWithTest(testFunc func() error) error {
	// 创建一个带超时的 context
	ctx, cancel := context.WithTimeout(context.Background(), n.timeout)
	defer cancel()

	// 在协程中发送测试消息
	errChan := make(chan error, 1)
	go func() {
		errChan <- testFunc()
	}()

	// 等待测试消息发送完成或超时
	select {
	case err := <-errChan:
		if err != nil {
			n.logger.Warn("测试消息发送失败",
				zap.String("notifier_zh", n.nameZh),
				zap.String("notifier_en", n.nameEn),
				zap.Error(err),
			)
			return err
		}
		n.logger.Info("测试消息发送成功",
			zap.String("notifier_zh", n.nameZh),
			zap.String("notifier_en", n.nameEn),
		)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("测试消息发送超时（%v）", n.timeout)
	}
}
