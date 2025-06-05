package monitor

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// TCPState TCP 连接状态
type TCPState struct {
	Established int // 已建立的连接
	Listen      int // 监听中的连接
	TimeWait    int // 等待关闭的连接
	SynRecv     int // 接收到 SYN 的连接
	CloseWait   int // 等待关闭的连接
	LastAck     int // 等待最后确认的连接
	SynSent     int // 已发送 SYN 的连接
	Closing     int // 正在关闭的连接
	FinWait1    int // 等待对方 FIN 的连接
	FinWait2    int // 等待连接关闭的连接
}

// TCPMonitor TCP 监控器
type TCPMonitor struct {
	BaseMonitor
}

// NewTCPMonitor 创建新的 TCP 监控器
func NewTCPMonitor(logger *zap.Logger, interval time.Duration, runMode string) *TCPMonitor {
	return &TCPMonitor{
		BaseMonitor: NewBaseMonitor("TCP监控", logger, interval, runMode),
	}
}

// Start 启动 TCP 监控
func (tm *TCPMonitor) Start() {
	tm.BaseMonitor.Start(tm.monitor)
}

// Stop 停止 TCP 监控
func (tm *TCPMonitor) Stop() {
	tm.BaseMonitor.Stop()
}

// monitor TCP 监控主循环
func (tm *TCPMonitor) monitor() {
	defer tm.Done()
	ticker := time.NewTicker(tm.GetInterval())
	defer ticker.Stop()

	for {
		if tm.IsStopped() {
			return
		}

		select {
		case <-tm.stopChan:
			return
		case <-ticker.C:
			state, err := tm.GetTCPState()
			if err != nil {
				tm.GetLogger().Error("获取 TCP 状态失败", zap.Error(err))
				continue
			}

			// 记录 TCP 状态
			tm.GetLogger().Info("TCP 连接状态统计",
				zap.Int("established", state.Established),
				zap.Int("listen", state.Listen),
				zap.Int("time_wait", state.TimeWait),
				zap.Int("syn_recv", state.SynRecv),
				zap.Int("close_wait", state.CloseWait),
				zap.Int("last_ack", state.LastAck),
				zap.Int("syn_sent", state.SynSent),
				zap.Int("closing", state.Closing),
				zap.Int("fin_wait1", state.FinWait1),
				zap.Int("fin_wait2", state.FinWait2),
			)
		}
	}
}

// GetTCPState 获取当前 TCP 连接状态
func (tm *TCPMonitor) GetTCPState() (*TCPState, error) {
	// 读取 /proc/net/tcp 文件
	content, err := ioutil.ReadFile("/proc/net/tcp")
	if err != nil {
		return nil, fmt.Errorf("读取 /proc/net/tcp 失败: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	state := &TCPState{}

	// 跳过标题行
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		// TCP 状态在第四列，是一个十六进制数
		stateHex := fields[3]
		stateNum, err := strconv.ParseInt(stateHex, 16, 64)
		if err != nil {
			continue
		}

		// 根据 TCP 状态码更新计数
		// 状态码参考: include/net/tcp_states.h
		switch stateNum {
		case 1:
			state.Established++
		case 2:
			state.SynSent++
		case 3:
			state.SynRecv++
		case 4:
			state.FinWait1++
		case 5:
			state.FinWait2++
		case 6:
			state.TimeWait++
		case 7:
			state.CloseWait++
		case 8:
			state.LastAck++
		case 9:
			state.Listen++
		case 10:
			state.Closing++
		}
	}

	return state, nil
}
