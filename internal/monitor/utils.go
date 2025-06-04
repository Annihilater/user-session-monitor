package monitor

import (
	"fmt"
	"time"
)

// 定义容量单位
const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
	TB = 1024 * GB
)

const (
	Bps  = 1
	KBps = 1024 * Bps
	MBps = 1024 * KBps
	GBps = 1024 * MBps
	TBps = 1024 * GBps
)

// formatBytes 将字节数转换为人类可读的格式
func formatBytes(bytes uint64) string {
	var (
		value float64
		unit  string
	)

	switch {
	case bytes >= TB:
		value = float64(bytes) / float64(TB)
		unit = "TB"
	case bytes >= GB:
		value = float64(bytes) / float64(GB)
		unit = "GB"
	case bytes >= MB:
		value = float64(bytes) / float64(MB)
		unit = "MB"
	case bytes >= KB:
		value = float64(bytes) / float64(KB)
		unit = "KB"
	default:
		value = float64(bytes)
		unit = "B"
	}

	return fmt.Sprintf("%.2f %s", value, unit)
}

// formatBytes 格式化字节大小为GB
func formatBytesToGB(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024 * 1024)
}

// formatUptime 格式化运行时间
func formatUptime(uptime time.Duration) string {
	days := int(uptime.Hours()) / 24
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60

	return fmt.Sprintf("%d天%d小时%d分钟%d秒", days, hours, minutes, seconds)
}

// formatSpeed 将速度转换为合适的单位
func formatSpeed(bytesPerSec float64) string {
	var unit string
	var value float64

	switch {
	case bytesPerSec >= TBps:
		unit = "TB/s"
		value = bytesPerSec / TBps
	case bytesPerSec >= GBps:
		unit = "GB/s"
		value = bytesPerSec / GBps
	case bytesPerSec >= MBps:
		unit = "MB/s"
		value = bytesPerSec / MBps
	case bytesPerSec >= KBps:
		unit = "KB/s"
		value = bytesPerSec / KBps
	default:
		unit = "B/s"
		value = bytesPerSec
	}

	return fmt.Sprintf("%.2f %s", value, unit)
}
