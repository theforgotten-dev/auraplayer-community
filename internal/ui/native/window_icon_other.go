//go:build !windows

package native

import "gioui.org/io/event"

type taskbarIcon struct{}

func (*taskbarIcon) handleEvent(event.Event, func(func())) error { return nil }
