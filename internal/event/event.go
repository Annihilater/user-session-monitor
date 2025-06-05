package event

import (
	"github.com/Annihilater/user-session-monitor/internal/types"
)

// Type 定义事件类型
type Type int

// Bus 事件总线
type Bus struct {
	eventChan chan types.Event
}

// NewBus 创建新的事件总线
func NewBus(bufferSize int) *Bus {
	return &Bus{
		eventChan: make(chan types.Event, bufferSize),
	}
}

// Publish 发布事件
func (eb *Bus) Publish(event types.Event) {
	eb.eventChan <- event
}

// Subscribe 订阅事件
func (eb *Bus) Subscribe() <-chan types.Event {
	return eb.eventChan
}
