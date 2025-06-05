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
