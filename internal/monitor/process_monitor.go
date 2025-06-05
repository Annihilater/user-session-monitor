package monitor

import (
	"sort"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
	"go.uber.org/zap"

	"github.com/Annihilater/user-session-monitor/internal/types"
)

// ProcessMonitor 进程监控器
type ProcessMonitor struct {
	BaseMonitor
}

// NewProcessMonitor 创建新的进程监控器
func NewProcessMonitor(logger *zap.Logger, interval time.Duration, runMode string) *ProcessMonitor {
	return &ProcessMonitor{
		BaseMonitor: NewBaseMonitor("进程监控", logger, interval, runMode),
	}
}

// Start 启动进程监控
func (pm *ProcessMonitor) Start() {
	pm.BaseMonitor.Start(pm.monitor)
}

// Stop 停止进程监控
func (pm *ProcessMonitor) Stop() {
	pm.BaseMonitor.Stop()
}

// getTopProcesses 获取 CPU 占用最高的进程
func (pm *ProcessMonitor) getTopProcesses(count int) ([]types.ProcessInfo, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	// 获取系统总内存
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	totalMem := memInfo.Total

	var processInfos []types.ProcessInfo
	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}

		command, err := p.Cmdline()
		if err != nil {
			command = "未知"
		}

		cpu, err := p.CPUPercent()
		if err != nil {
			continue
		}

		mem, err := p.MemoryInfo()
		if err != nil {
			continue
		}

		username, err := p.Username()
		if err != nil {
			username = "未知"
		}

		createTime, err := p.CreateTime()
		if err != nil {
			createTime = 0
		}

		// 计算内存使用百分比
		memPercent := float32(mem.RSS) / float32(totalMem) * 100

		processInfos = append(processInfos, types.ProcessInfo{
			PID:           p.Pid,
			Name:          name,
			Command:       command,
			CPUPercent:    cpu,
			MemoryUsage:   mem.RSS,
			MemoryPercent: memPercent,
			Username:      username,
			CreateTime:    time.Unix(createTime/1000, 0),
		})
	}

	// 按 CPU 使用率排序
	sort.Slice(processInfos, func(i, j int) bool {
		return processInfos[i].CPUPercent > processInfos[j].CPUPercent
	})

	// 返回前 N 个进程
	if len(processInfos) > count {
		processInfos = processInfos[:count]
	}

	return processInfos, nil
}

// monitor 进程监控主循环
func (pm *ProcessMonitor) monitor() {
	defer pm.Done()
	ticker := time.NewTicker(pm.GetInterval())
	defer ticker.Stop()

	for {
		if pm.IsStopped() {
			return
		}

		select {
		case <-pm.stopChan:
			return
		case <-ticker.C:
			// 获取进程总数
			processes, err := process.Processes()
			if err != nil {
				pm.GetLogger().Error("获取进程列表失败", zap.Error(err))
				continue
			}

			// 获取 CPU 占用最高的 10 个进程
			topProcesses, err := pm.getTopProcesses(10)
			if err != nil {
				pm.GetLogger().Error("获取 TOP 进程失败", zap.Error(err))
				continue
			}

			// 记录进程信息
			pm.GetLogger().Info("进程状态",
				zap.Int("进程总数", len(processes)),
				zap.Int("TOP进程数", len(topProcesses)),
			)

			// 记录每个 TOP 进程的详细信息
			for i, proc := range topProcesses {
				pm.GetLogger().Info("TOP进程详情",
					zap.Int("proc_rank", i+1),
					zap.Int32("proc_pid", proc.PID),
					zap.String("proc_name", proc.Name),
					zap.String("proc_command", proc.Command),
					zap.String("proc_cpu_percent", formatPercent(proc.CPUPercent)),
					zap.String("proc_memory_usage", formatBytes(proc.MemoryUsage)),
					zap.String("proc_memory_percent", formatPercent(float64(proc.MemoryPercent))),
					zap.String("proc_user", proc.Username),
					zap.Time("proc_create_time", proc.CreateTime),
				)
			}
		}
	}
}
