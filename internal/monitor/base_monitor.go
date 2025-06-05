package monitor

import (
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
)

// BaseMonitor 基础监控器，包含所有监控器共有的字段和方法
type BaseMonitor struct {
	name     string         // 监控器名称
	logger   *zap.Logger    // 日志器
	interval time.Duration  // 监控间隔
	stopChan chan struct{}  // 停止信号
	wg       sync.WaitGroup // 等待组
	runMode  string         // 运行模式：thread 或 goroutine
}

// NewBaseMonitor 创建基础监控器
func NewBaseMonitor(name string, logger *zap.Logger, interval time.Duration, runMode string) BaseMonitor {
	return BaseMonitor{
		name:     name,
		logger:   logger,
		interval: interval,
		stopChan: make(chan struct{}),
		runMode:  runMode,
	}
}

// Start 启动监控，需要传入具体的监控函数
func (b *BaseMonitor) Start(monitorFunc func()) {
	b.wg.Add(1)
	b.logger.Info("启动监控",
		zap.String("monitor", b.name),
		zap.String("run_mode", b.runMode),
	)

	if b.runMode == "thread" {
		go func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			monitorFunc()
		}()
	} else {
		go monitorFunc()
	}
}

// Stop 停止监控
func (b *BaseMonitor) Stop() {
	close(b.stopChan)
	b.wg.Wait()
}

// IsStopped 检查是否收到停止信号
func (b *BaseMonitor) IsStopped() bool {
	select {
	case <-b.stopChan:
		return true
	default:
		return false
	}
}

// Done 标记监控完成
func (b *BaseMonitor) Done() {
	b.wg.Done()
}

// GetInterval 获取监控间隔
func (b *BaseMonitor) GetInterval() time.Duration {
	return b.interval
}

// GetLogger 获取日志器
func (b *BaseMonitor) GetLogger() *zap.Logger {
	return b.logger
}
