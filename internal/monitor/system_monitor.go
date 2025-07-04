package monitor

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

// SystemMonitor 系统监控器
type SystemMonitor struct {
	BaseMonitor
	diskPaths []string // 要监控的磁盘路径列表
}

// NewSystemMonitor 创建新的系统监控器
func NewSystemMonitor(logger *zap.Logger, interval time.Duration, diskPaths []string, runMode string) *SystemMonitor {
	if len(diskPaths) == 0 {
		diskPaths = []string{"/"} // 默认监控根目录
	}
	return &SystemMonitor{
		BaseMonitor: NewBaseMonitor("系统监控", logger, interval, runMode),
		diskPaths:   diskPaths,
	}
}

// Start 启动系统监控
func (sm *SystemMonitor) Start() {
	sm.BaseMonitor.Start(sm.monitor)
}

// Stop 停止系统监控
func (sm *SystemMonitor) Stop() {
	sm.BaseMonitor.Stop()
}

// monitor 系统监控主循环
func (sm *SystemMonitor) monitor() {
	defer sm.Done()
	ticker := time.NewTicker(sm.GetInterval())
	defer ticker.Stop()

	for {
		if sm.IsStopped() {
			return
		}

		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			// 获取 CPU 使用率
			cpuPercent, err := cpu.Percent(0, false)
			if err != nil {
				sm.GetLogger().Error("获取CPU使用率失败", zap.Error(err))
			} else if len(cpuPercent) > 0 {
				sm.GetLogger().Info("CPU状态",
					zap.String("usage", fmt.Sprintf("%.2f%%", cpuPercent[0])),
				)
			}

			// 获取内存使用情况
			memInfo, err := mem.VirtualMemory()
			if err != nil {
				sm.GetLogger().Error("获取内存信息失败", zap.Error(err))
			} else {
				// 计算 Swap 使用量和使用率
				swapUsed := memInfo.SwapTotal - memInfo.SwapFree
				swapUsedPercent := float64(0)
				if memInfo.SwapTotal > 0 {
					swapUsedPercent = float64(swapUsed) / float64(memInfo.SwapTotal) * 100
				}

				sm.GetLogger().Info("内存状态",
					// 物理内存指标
					zap.String("usage", fmt.Sprintf("%.2f%%", memInfo.UsedPercent)),
					zap.String("total", formatBytes(memInfo.Total)),
					zap.String("used", formatBytes(memInfo.Used)),
					zap.String("available", formatBytes(memInfo.Available)),
					// 虚拟内存（Swap）指标
					zap.String("swap_total", formatBytes(memInfo.SwapTotal)),
					zap.String("swap_used", formatBytes(swapUsed)),
					zap.String("swap_free", formatBytes(memInfo.SwapFree)),
					zap.String("swap_usage", fmt.Sprintf("%.2f%%", swapUsedPercent)),
				)
			}

			// 获取磁盘使用情况
			for _, path := range sm.diskPaths {
				usage, err := disk.Usage(path)
				if err != nil {
					sm.GetLogger().Error("获取磁盘使用情况失败",
						zap.String("path", path),
						zap.Error(err),
					)
					continue
				}
				sm.GetLogger().Info("磁盘状态",
					zap.String("path", path),
					zap.String("usage", fmt.Sprintf("%.2f%%", usage.UsedPercent)),
					zap.String("total", formatBytes(usage.Total)),
					zap.String("used", formatBytes(usage.Used)),
					zap.String("free", formatBytes(usage.Free)),
				)
			}

			// 获取系统运行时间
			hostInfo, err := host.Info()
			if err != nil {
				sm.GetLogger().Error("获取主机信息失败", zap.Error(err))
			} else {
				uptime := time.Duration(hostInfo.Uptime) * time.Second
				sm.GetLogger().Info("系统运行时间",
					zap.String("uptime", formatUptime(uptime)),
				)
			}

			// 获取系统负载
			loadInfo, err := load.Avg()
			if err != nil {
				sm.GetLogger().Error("获取系统负载失败", zap.Error(err))
			} else {
				sm.GetLogger().Info("系统负载",
					zap.Float64("load1", loadInfo.Load1),
					zap.Float64("load5", loadInfo.Load5),
					zap.Float64("load15", loadInfo.Load15),
				)
			}
		}
	}
}
