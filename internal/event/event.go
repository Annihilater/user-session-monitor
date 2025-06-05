package event

import (
	"sync"

	"github.com/Annihilater/user-session-monitor/internal/types"
)

// Type 定义事件类型
type Type int

// Bus 事件总线
type Bus struct {
	subscribers []chan types.Event
	mu          sync.RWMutex
}

// NewBus 创建新的事件总线
func NewBus(bufferSize int) *Bus {
	return &Bus{
		subscribers: make([]chan types.Event, 0),
	}
}

// Publish 发布事件
func (eb *Bus) Publish(event types.Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// 向所有订阅者发送事件
	for _, ch := range eb.subscribers {
		// 使用非阻塞发送，避免一个订阅者阻塞其他订阅者
		select {
		case ch <- event:
		default:
			// 如果通道已满，跳过这个订阅者
		}
	}
}

// Subscribe 订阅事件
func (eb *Bus) Subscribe() <-chan types.Event {
	ch := make(chan types.Event, 100) // 为每个订阅者创建一个带缓冲的通道

	eb.mu.Lock()
	eb.subscribers = append(eb.subscribers, ch)
	eb.mu.Unlock()

	return ch
}

// Unsubscribe 取消订阅
func (eb *Bus) Unsubscribe(ch <-chan types.Event) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for i, subCh := range eb.subscribers {
		if subCh == ch {
			// 从订阅者列表中移除
			eb.subscribers = append(eb.subscribers[:i], eb.subscribers[i+1:]...)
			close(subCh)
			break
		}
	}
}
