package monitor

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

// HardwareMonitor 硬件信息监控器
type HardwareMonitor struct {
	logger    *zap.Logger
	interval  time.Duration
	stopChan  chan struct{}
	wg        sync.WaitGroup
	diskPaths []string
	runMode   string // 运行模式：thread 或 goroutine
}

// NewHardwareMonitor 创建新的硬件信息监控器
func NewHardwareMonitor(logger *zap.Logger, interval time.Duration, diskPaths []string, runMode string) *HardwareMonitor {
	if len(diskPaths) == 0 {
		diskPaths = []string{"/"}
	}
	return &HardwareMonitor{
		logger:    logger,
		interval:  interval,
		stopChan:  make(chan struct{}),
		diskPaths: diskPaths,
		runMode:   runMode,
	}
}

// Start 启动硬件信息监控
func (hm *HardwareMonitor) Start() {
	hm.wg.Add(1)
	hm.logger.Info("启动硬件监控",
		zap.String("run_mode", hm.runMode),
	)
	if hm.runMode == "thread" {
		go func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			hm.monitorHardware()
		}()
	} else {
		go hm.monitorHardware()
	}
}

// Stop 停止硬件信息监控
func (hm *HardwareMonitor) Stop() {
	close(hm.stopChan)
	hm.wg.Wait()
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
		defer resp.Body.Close()

		ip, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		// 清理IP地址字符串
		ipStr := strings.TrimSpace(string(ip))
		if ipStr != "" {
			return ipStr
		}
	}

	return "未知"
}

// monitorHardware 监控硬件信息
func (hm *HardwareMonitor) monitorHardware() {
	defer hm.wg.Done()
	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	// 立即执行一次
	hm.collectAndLogHardwareInfo()

	for {
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
