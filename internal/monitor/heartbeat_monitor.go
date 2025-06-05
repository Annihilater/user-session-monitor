package monitor

import (
	"time"

	"go.uber.org/zap"
)

// HeartbeatMonitor 心跳监控器
type HeartbeatMonitor struct {
	BaseMonitor
}

// NewHeartbeatMonitor 创建新的心跳监控器
func NewHeartbeatMonitor(logger *zap.Logger, interval time.Duration, runMode string) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		BaseMonitor: NewBaseMonitor("心跳监控", logger, interval, runMode),
	}
}

// Start 启动心跳监控
func (hm *HeartbeatMonitor) Start() {
	hm.BaseMonitor.Start(hm.monitor)
}

// Stop 停止心跳监控
func (hm *HeartbeatMonitor) Stop() {
	hm.BaseMonitor.Stop()
}

// monitor 心跳监控主循环
func (hm *HeartbeatMonitor) monitor() {
	defer hm.Done()
	ticker := time.NewTicker(hm.GetInterval())
	defer ticker.Stop()

	// 记录启动时间
	startTime := time.Now()

	for {
		if hm.IsStopped() {
			return
		}

		select {
		case <-hm.stopChan:
			return
		case <-ticker.C:
			uptime := time.Since(startTime)
			hm.GetLogger().Info("监控程序心跳",
				zap.Duration("uptime", uptime),
				zap.Duration("interval", hm.GetInterval()),
			)
		}
	}
}
