package monitor

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

// HardwareMonitor 硬件信息监控器
type HardwareMonitor struct {
	BaseMonitor
	diskPaths []string
}

// NewHardwareMonitor 创建新的硬件信息监控器
func NewHardwareMonitor(logger *zap.Logger, interval time.Duration, diskPaths []string, runMode string) *HardwareMonitor {
	if len(diskPaths) == 0 {
		diskPaths = []string{"/"}
	}
	return &HardwareMonitor{
		BaseMonitor: NewBaseMonitor("硬件监控", logger, interval, runMode),
		diskPaths:   diskPaths,
	}
}

// Start 启动硬件信息监控
func (hm *HardwareMonitor) Start() {
	hm.BaseMonitor.Start(hm.monitorHardware)
}

// Stop 停止硬件信息监控
func (hm *HardwareMonitor) Stop() {
	hm.BaseMonitor.Stop()
}

// getPublicIP 获取公网IP地址
func (hm *HardwareMonitor) getPublicIP() string {
	// 使用多个IP查询服务，提高可靠性
	ipServices := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	}

	for _, service := range ipServices {
		client := http.Client{
			Timeout: 5 * time.Second,
		}
		resp, err := client.Get(service)
		if err != nil {
			continue
		}

		// 读取响应体并确保关闭
		ip, err := func() (string, error) {
			defer func() {
				if closeErr := resp.Body.Close(); closeErr != nil {
					hm.GetLogger().Error("关闭响应体失败",
						zap.String("service", service),
						zap.Error(closeErr),
					)
				}
			}()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(body)), nil
		}()

		// 如果获取成功且IP不为空，则返回
		if err == nil && ip != "" {
			return ip
		}
	}

	return "未知"
}

// monitorHardware 监控硬件信息
func (hm *HardwareMonitor) monitorHardware() {
	defer hm.Done()
	ticker := time.NewTicker(hm.GetInterval())
	defer ticker.Stop()

	// 立即执行一次
	hm.collectAndLogHardwareInfo()

	for {
		if hm.IsStopped() {
			return
		}

		select {
		case <-hm.stopChan:
			return
		case <-ticker.C:
			hm.collectAndLogHardwareInfo()
		}
	}
}

// collectAndLogHardwareInfo 收集并记录硬件信息
func (hm *HardwareMonitor) collectAndLogHardwareInfo() {
	// 获取CPU信息
	cpuInfo, err := cpu.Info()
	if err != nil {
		hm.logger.Error("获取CPU信息失败", zap.Error(err))
		return
	}

	var cpuModel string
	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	} else {
		cpuModel = "未知"
	}

	// 获取CPU核心数
	physicalCores, err := cpu.Counts(false) // false 表示只获取物理核心数
	if err != nil {
		hm.logger.Error("获取CPU核心数失败", zap.Error(err))
		return
	}

	logicalCores, err := cpu.Counts(true) // true 表示获取逻辑核心数（包括超线程）
	if err != nil {
		hm.logger.Error("获取CPU逻辑核心数失败", zap.Error(err))
		return
	}

	// 获取内存信息
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		hm.logger.Error("获取内存信息失败", zap.Error(err))
		return
	}

	// 获取主机信息
	hostInfo, err := host.Info()
	if err != nil {
		hm.logger.Error("获取主机信息失败", zap.Error(err))
		return
	}

	// 获取公网IP
	publicIP := hm.getPublicIP()

	// 获取磁盘信息
	var totalDiskGB float64
	for _, path := range hm.diskPaths {
		usage, err := disk.Usage(path)
		if err != nil {
			hm.logger.Error("获取磁盘信息失败",
				zap.String("path", path),
				zap.Error(err),
			)
			continue
		}
		totalDiskGB += formatBytesToGB(usage.Total)
	}

	// 记录硬件信息
	hm.logger.Info("硬件信息",
		// CPU信息
		zap.String("cpu_model", cpuModel),
		zap.String("cpu_arch", hostInfo.KernelArch),
		zap.String("physical_cpu_cores", fmt.Sprintf("%d 核", physicalCores)),
		zap.String("logical_cpu_cores", fmt.Sprintf("%d 核", logicalCores)),
		// 内存信息
		zap.String("total_memory", fmt.Sprintf("%.2f GB", formatBytesToGB(memInfo.Total))),
		// 磁盘信息
		zap.String("total_disk", fmt.Sprintf("%.2f GB", totalDiskGB)),
		// 网络信息
		zap.String("public_ip", publicIP),
		// 系统信息
		zap.String("os_platform", hostInfo.Platform),
		zap.String("os_family", hostInfo.PlatformFamily),
		zap.String("os_version", hostInfo.PlatformVersion),
		zap.String("kernel_version", hostInfo.KernelVersion),
	)
}
