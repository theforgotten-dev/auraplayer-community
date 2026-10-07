//go:build !windows

package native

import "gioui.org/io/event"

type fileDropTarget struct{}

func (*fileDropTarget) handleEvent(event.Event, func([]string), func(func())) error { return nil }
