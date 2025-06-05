package types

import "time"

// ServerInfo 服务器信息
type ServerInfo struct {
	Hostname string
	IP       string
	OSType   string
}

// LoginRecord 存储单个登录会话的详细信息
type LoginRecord struct {
	Username      string    // 用户名
	Ip            string    // 登录源 IP
	Port          string    // 登录源端口
	LastLoginTime time.Time // 最近一次登录时间
}

// Event 定义事件结构
type Event struct {
	Type       EventType
	Username   string
	IP         string
	Port       string
	Timestamp  time.Time
	ServerInfo *ServerInfo
}

// EventType 定义事件类型
type EventType int

const (
	EventTypeLogin EventType = iota
	EventTypeLogout
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

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID           int32
	Name          string
	Command       string
	CPUPercent    float64
	MemoryUsage   uint64
	MemoryPercent float32
	Username      string
	CreateTime    time.Time
}

// NotifyMessage 通知消息结构
type NotifyMessage struct {
	MsgType string                 `json:"msg_type"`
	Content map[string]interface{} `json:"content"`
}
