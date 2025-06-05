package notify

import "fmt"

// NotifyError 定义通知错误的结构
type NotifyError struct {
	Provider string
	Message  string
	Err      error
}

func (e *NotifyError) Error() string {
	return fmt.Sprintf("%s: %s: %v", e.Provider, e.Message, e.Err)
}

// NewNotifyError 创建一个新的通知错误
func NewNotifyError(provider, message string, err error) error {
	return &NotifyError{
		Provider: provider,
		Message:  message,
		Err:      err,
	}
}
