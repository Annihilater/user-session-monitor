package monitor

import (
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
)

// HeartbeatMonitor 心跳监控器
type HeartbeatMonitor struct {
	logger   *zap.Logger
	interval time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
	runMode  string // 运行模式：thread 或 goroutine
}

// NewHeartbeatMonitor 创建新的心跳监控器
func NewHeartbeatMonitor(logger *zap.Logger, interval time.Duration, runMode string) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		logger:   logger,
		interval: interval,
		stopChan: make(chan struct{}),
		runMode:  runMode,
	}
}

// Start 启动心跳监控
func (hm *HeartbeatMonitor) Start() {
	hm.wg.Add(1)
	hm.logger.Info("启动心跳监控",
		zap.String("run_mode", hm.runMode),
	)
	if hm.runMode == "thread" {
		go func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			hm.monitor()
		}()
	} else {
		go hm.monitor()
	}
}

// Stop 停止心跳监控
func (hm *HeartbeatMonitor) Stop() {
	close(hm.stopChan)
	hm.wg.Wait()
}

// monitor 心跳监控主循环
func (hm *HeartbeatMonitor) monitor() {
	defer hm.wg.Done()
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
