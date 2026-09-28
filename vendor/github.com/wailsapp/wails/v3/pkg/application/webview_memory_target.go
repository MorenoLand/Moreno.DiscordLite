package application

import "errors"

type MemoryUsageTargetLevel uint32

const (
	MemoryUsageTargetLevelNormal MemoryUsageTargetLevel = iota
	MemoryUsageTargetLevelLow
)

type memoryUsageTargetLevelSetter interface {
	setMemoryUsageTargetLevel(MemoryUsageTargetLevel) error
}

func (w *WebviewWindow) SetMemoryUsageTargetLevel(level MemoryUsageTargetLevel) error {
	if level > MemoryUsageTargetLevelLow {
		return errors.New("invalid WebView2 memory target level")
	}
	if w.impl == nil || w.isDestroyed() {
		return nil
	}
	setter, ok := w.impl.(memoryUsageTargetLevelSetter)
	if !ok {
		return nil
	}
	return InvokeSyncWithError(func() error { return setter.setMemoryUsageTargetLevel(level) })
}
