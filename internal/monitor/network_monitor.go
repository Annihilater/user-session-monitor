package monitor

import (
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/net"
	"go.uber.org/zap"
)

// NetworkMonitor 网络监控器
type NetworkMonitor struct {
	logger   *zap.Logger
	interval time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
	runMode  string // 运行模式：thread 或 goroutine

	// 用于计算速度的上一次统计数据
	lastStats net.IOCountersStat
	lastTime  time.Time
}

// NewNetworkMonitor 创建新的网络监控器
func NewNetworkMonitor(logger *zap.Logger, interval time.Duration, runMode string) *NetworkMonitor {
	return &NetworkMonitor{
		logger:   logger,
		interval: interval,
		stopChan: make(chan struct{}),
		runMode:  runMode,
	}
}

// Start 启动网络监控
func (nm *NetworkMonitor) Start() {
	nm.wg.Add(1)
	nm.logger.Info("启动网络监控",
		zap.String("run_mode", nm.runMode),
	)
	if nm.runMode == "thread" {
		go func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			nm.monitor()
		}()
	} else {
		go nm.monitor()
	}
}

// Stop 停止网络监控
func (nm *NetworkMonitor) Stop() {
	close(nm.stopChan)
	nm.wg.Wait()
}

// monitor 网络监控主循环
func (nm *NetworkMonitor) monitor() {
	defer nm.wg.Done()
	ticker := time.NewTicker(nm.interval)
	defer ticker.Stop()

	// 初始化上一次的统计数据
	stats, err := net.IOCounters(false)
	if err != nil {
		nm.logger.Error("获取网络统计信息失败", zap.Error(err))
		return
	}
	if len(stats) > 0 {
		nm.lastStats = stats[0]
		nm.lastTime = time.Now()
	}

	for {
		select {
		case <-nm.stopChan:
			return
		case <-ticker.C:
			stats, err := net.IOCounters(false)
			if err != nil {
				nm.logger.Error("获取网络统计信息失败", zap.Error(err))
				continue
			}
			if len(stats) == 0 {
				continue
			}

			currentStats := stats[0]
			currentTime := time.Now()
			timeDiff := currentTime.Sub(nm.lastTime).Seconds()

			// 计算速度（字节/秒）
			uploadSpeed := float64(currentStats.BytesSent-nm.lastStats.BytesSent) / timeDiff
			downloadSpeed := float64(currentStats.BytesRecv-nm.lastStats.BytesRecv) / timeDiff

			// 更新记录
			nm.lastStats = currentStats
			nm.lastTime = currentTime

			// 记录网络状态
			nm.logger.Info("网络状态",
				zap.String("upload_speed", formatSpeed(uploadSpeed)),
				zap.String("download_speed", formatSpeed(downloadSpeed)),
				zap.String("total_upload", formatBytes(currentStats.BytesSent)),
				zap.String("total_download", formatBytes(currentStats.BytesRecv)),
				zap.String("packets_sent", formatBytes(currentStats.PacketsSent)),
				zap.String("packets_recv", formatBytes(currentStats.PacketsRecv)),
			)
		}
	}
}
