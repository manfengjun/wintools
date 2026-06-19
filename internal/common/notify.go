package common

import (
	"context"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ── 通知类型 ──

type NotifyType string

const (
	NotifyInfo    NotifyType = "info"
	NotifyWarn    NotifyType = "warning"
	NotifyError   NotifyType = "error"
	NotifySuccess NotifyType = "success"
)

// NotifyPayload 前端收到的通知结构
type NotifyPayload struct {
	Type    NotifyType `json:"type"`
	Title   string     `json:"title"`
	Message string     `json:"message"`
}

// Notify 发送通知到前端（通过 EventsEmit）
func Notify(ctx context.Context, nType NotifyType, title, message string) {
	if ctx == nil {
		return
	}
	wailsRuntime.EventsEmit(ctx, "notify", NotifyPayload{
		Type:    nType,
		Title:   title,
		Message: message,
	})
}

// SendInfo 快捷方式
func SendInfo(ctx context.Context, title, message string) {
	Notify(ctx, NotifyInfo, title, message)
	Info("[notify] %s: %s", title, message)
}

// SendWarn 快捷方式
func SendWarn(ctx context.Context, title, message string) {
	Notify(ctx, NotifyWarn, title, message)
	Warn("[notify] %s: %s", title, message)
}

// SendError 快捷方式
func SendError(ctx context.Context, title, message string) {
	Notify(ctx, NotifyError, title, message)
	Error("[notify] %s: %s", title, message)
}

// SendSuccess 快捷方式
func SendSuccess(ctx context.Context, title, message string) {
	Notify(ctx, NotifySuccess, title, message)
	Info("[notify] %s: %s", title, message)
}
