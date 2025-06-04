package monitor

import (
	"time"

	"go.uber.org/zap"
)

// HeartbeatMonitor 心跳监控器
type HeartbeatMonitor struct {
	logger   *zap.Logger
	interval time.Duration
	stopChan chan struct{}
}

// NewHeartbeatMonitor 创建新的心跳监控器
func NewHeartbeatMonitor(logger *zap.Logger, interval time.Duration) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		logger:   logger,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start 启动心跳监控
func (hm *HeartbeatMonitor) Start() {
	go hm.monitor()
}

// Stop 停止心跳监控
func (hm *HeartbeatMonitor) Stop() {
	close(hm.stopChan)
}

// monitor 心跳监控主循环
func (hm *HeartbeatMonitor) monitor() {
	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	// 记录启动时间
	startTime := time.Now()

	for {
		select {
		case <-hm.stopChan:
			return
		case <-ticker.C:
			uptime := time.Since(startTime)
			hm.logger.Info("监控程序心跳",
				zap.Duration("uptime", uptime),
				zap.Duration("interval", hm.interval),
			)
		}
	}
}
