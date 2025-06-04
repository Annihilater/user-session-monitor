package monitor

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

// 定义容量单位
const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
	TB = 1024 * GB
)

// formatSize 格式化字节大小为人类可读格式
func formatSize(bytes uint64) (float64, string) {
	switch {
	case bytes >= TB:
		return float64(bytes) / TB, "TB"
	case bytes >= GB:
		return float64(bytes) / GB, "GB"
	case bytes >= MB:
		return float64(bytes) / MB, "MB"
	case bytes >= KB:
		return float64(bytes) / KB, "KB"
	default:
		return float64(bytes), "B"
	}
}

// SystemMonitor 系统资源监控器
type SystemMonitor struct {
	logger    *zap.Logger
	interval  time.Duration
	stopChan  chan struct{}
	diskPaths []string // 要监控的磁盘路径列表
}

// NewSystemMonitor 创建新的系统资源监控器
func NewSystemMonitor(logger *zap.Logger, interval time.Duration, diskPaths []string) *SystemMonitor {
	if len(diskPaths) == 0 {
		diskPaths = []string{"/"} // 默认监控根目录
	}
	return &SystemMonitor{
		logger:    logger,
		interval:  interval,
		stopChan:  make(chan struct{}),
		diskPaths: diskPaths,
	}
}

// Start 启动系统资源监控
func (sm *SystemMonitor) Start() {
	go sm.monitorCPU()
	go sm.monitorMemory()
	go sm.monitorDisk()
}

// Stop 停止系统资源监控
func (sm *SystemMonitor) Stop() {
	close(sm.stopChan)
}

// monitorCPU CPU 使用率监控
func (sm *SystemMonitor) monitorCPU() {
	ticker := time.NewTicker(sm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			percentage, err := cpu.Percent(0, false) // false 表示获取总体 CPU 使用率
			if err != nil {
				sm.logger.Error("获取CPU使用率失败", zap.Error(err))
				continue
			}
			if len(percentage) > 0 {
				sm.logger.Info("CPU使用率",
					zap.Float64("percentage", percentage[0]),
					zap.String("unit", "%"),
				)
			}
		}
	}
}

// monitorMemory 内存使用率监控
func (sm *SystemMonitor) monitorMemory() {
	ticker := time.NewTicker(sm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			v, err := mem.VirtualMemory()
			if err != nil {
				sm.logger.Error("获取内存使用率失败", zap.Error(err))
				continue
			}

			// 格式化内存大小
			totalSize, totalUnit := formatSize(v.Total)
			usedSize, usedUnit := formatSize(v.Used)
			freeSize, freeUnit := formatSize(v.Free)

			sm.logger.Info("内存使用情况",
				zap.Float64("used_percentage", v.UsedPercent),
				zap.String("percentage_unit", "%"),
				zap.Float64("total", totalSize),
				zap.String("total_unit", totalUnit),
				zap.Float64("used", usedSize),
				zap.String("used_unit", usedUnit),
				zap.Float64("free", freeSize),
				zap.String("free_unit", freeUnit),
			)
		}
	}
}

// monitorDisk 磁盘使用率监控
func (sm *SystemMonitor) monitorDisk() {
	ticker := time.NewTicker(sm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			for _, path := range sm.diskPaths {
				usage, err := disk.Usage(path)
				if err != nil {
					sm.logger.Error("获取磁盘使用率失败",
						zap.String("path", path),
						zap.Error(err),
					)
					continue
				}

				// 格式化磁盘大小
				totalSize, totalUnit := formatSize(usage.Total)
				usedSize, usedUnit := formatSize(usage.Used)
				freeSize, freeUnit := formatSize(usage.Free)

				sm.logger.Info("磁盘使用情况",
					zap.String("path", path),
					zap.Float64("used_percentage", usage.UsedPercent),
					zap.String("percentage_unit", "%"),
					zap.Float64("total", totalSize),
					zap.String("total_unit", totalUnit),
					zap.Float64("used", usedSize),
					zap.String("used_unit", usedUnit),
					zap.Float64("free", freeSize),
					zap.String("free_unit", freeUnit),
				)
			}
		}
	}
}
