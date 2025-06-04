package monitor

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

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
			sm.logger.Info("内存使用情况",
				zap.Float64("used_percentage", v.UsedPercent),
				zap.String("unit", "%"),
				zap.Uint64("total", v.Total),
				zap.Uint64("used", v.Used),
				zap.Uint64("free", v.Free),
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
				sm.logger.Info("磁盘使用情况",
					zap.String("path", path),
					zap.Float64("used_percentage", usage.UsedPercent),
					zap.String("unit", "%"),
					zap.Uint64("total", usage.Total),
					zap.Uint64("used", usage.Used),
					zap.Uint64("free", usage.Free),
				)
			}
		}
	}
}
