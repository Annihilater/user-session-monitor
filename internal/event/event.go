package event

import (
	"time"

	"github.com/Annihilater/user-session-monitor/internal/types"
)

// EventType 定义事件类型
type EventType int

const (
	EventTypeLogin EventType = iota
	EventTypeLogout
)

// Event 定义事件结构
type Event struct {
	Type       EventType
	Username   string
	IP         string
	Port       string
	Timestamp  time.Time
	ServerInfo *types.ServerInfo
}

// EventBus 事件总线
type EventBus struct {
	eventChan chan types.Event
}

// NewEventBus 创建新的事件总线
func NewEventBus(bufferSize int) *EventBus {
	return &EventBus{
		eventChan: make(chan types.Event, bufferSize),
	}
}

// Publish 发布事件
func (eb *EventBus) Publish(event types.Event) {
	eb.eventChan <- event
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe() <-chan types.Event {
	return eb.eventChan
}
