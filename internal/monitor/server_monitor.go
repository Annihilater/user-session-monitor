package monitor

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Annihilater/user-session-monitor/internal/types"
	"go.uber.org/zap"
)

// ServerMonitor 服务器信息监控器
type ServerMonitor struct {
	BaseMonitor
}

// NewServerMonitor 创建新的服务器信息监控器
func NewServerMonitor(logger *zap.Logger, interval time.Duration, runMode string) *ServerMonitor {
	return &ServerMonitor{
		BaseMonitor: NewBaseMonitor("服务器监控", logger, interval, runMode),
	}
}

// Start 启动服务器信息监控
func (sm *ServerMonitor) Start() {
	sm.BaseMonitor.Start(sm.monitor)
}

// Stop 停止服务器信息监控
func (sm *ServerMonitor) Stop() {
	sm.BaseMonitor.Stop()
}

// monitor 服务器信息监控主循环
func (sm *ServerMonitor) monitor() {
	defer sm.Done()
	ticker := time.NewTicker(sm.GetInterval())
	defer ticker.Stop()

	// 立即执行一次
	sm.collectAndLogServerInfo()

	for {
		if sm.IsStopped() {
			return
		}

		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.collectAndLogServerInfo()
		}
	}
}

// collectAndLogServerInfo 收集并记录服务器信息
func (sm *ServerMonitor) collectAndLogServerInfo() {
	serverInfo, err := sm.getServerInfo()
	if err != nil {
		sm.GetLogger().Error("获取服务器信息失败", zap.Error(err))
		return
	}

	// 记录服务器信息
	sm.GetLogger().Info("服务器信息",
		zap.String("hostname", serverInfo.Hostname),
		zap.String("ip", serverInfo.IP),
		zap.String("os_type", serverInfo.OSType),
	)
}

// getServerInfo 获取服务器信息
func (sm *ServerMonitor) getServerInfo() (*types.ServerInfo, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("获取主机名失败: %v", err)
	}

	// 获取非回环IP地址
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("获取网络接口地址失败: %v", err)
	}

	var ip string
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				break
			}
		}
	}

	if ip == "" {
		return nil, fmt.Errorf("未找到有效的IP地址")
	}

	// 获取操作系统类型
	osType, err := detectOSType()
	if err != nil {
		osType = "未知"
	}

	return &types.ServerInfo{
		Hostname: hostname,
		IP:       ip,
		OSType:   osType,
	}, nil
}
