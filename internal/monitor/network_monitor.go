package monitor

import (
	"time"

	"github.com/shirou/gopsutil/v3/net"
	"go.uber.org/zap"
)

// NetworkMonitor 网络监控器
type NetworkMonitor struct {
	logger   *zap.Logger
	interval time.Duration
	stopChan chan struct{}

	// 用于计算速度的上一次统计数据
	lastStats net.IOCountersStat
	lastTime  time.Time
}

// NewNetworkMonitor 创建新的网络监控器
func NewNetworkMonitor(logger *zap.Logger, interval time.Duration) *NetworkMonitor {
	return &NetworkMonitor{
		logger:   logger,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start 启动网络监控
func (nm *NetworkMonitor) Start() {
	go nm.monitor()
}

// Stop 停止网络监控
func (nm *NetworkMonitor) Stop() {
	close(nm.stopChan)
}

// monitor 网络监控主循环
func (nm *NetworkMonitor) monitor() {
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
				zap.String("当前上传速度", formatSpeed(uploadSpeed)),
				zap.String("当前下载速度", formatSpeed(downloadSpeed)),
				zap.String("总上传量", formatBytes(currentStats.BytesSent)),
				zap.String("总下载量", formatBytes(currentStats.BytesRecv)),
				zap.String("上传包数", formatBytes(currentStats.PacketsSent)),
				zap.String("下载包数", formatBytes(currentStats.PacketsRecv)),
			)
		}
	}
}
