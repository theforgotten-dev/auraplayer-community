//go:build windows

package native

import (
	"fmt"
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/event"
	"golang.org/x/sys/windows"
)

const (
	wmDropFiles    = 0x0233
	dropSubclassID = 0x41555241 // "AURA"
)

var (
	shell32Drop         = windows.NewLazySystemDLL("shell32.dll")
	comctl32Drop        = windows.NewLazySystemDLL("comctl32.dll")
	procDragAcceptFiles = shell32Drop.NewProc("DragAcceptFiles")
	procDragQueryFileW  = shell32Drop.NewProc("DragQueryFileW")
	procDragFinish      = shell32Drop.NewProc("DragFinish")
	procSetSubclass     = comctl32Drop.NewProc("SetWindowSubclass")
	procRemoveSubclass  = comctl32Drop.NewProc("RemoveWindowSubclass")
	procDefSubclass     = comctl32Drop.NewProc("DefSubclassProc")
)

type fileDropTarget struct {
	hwnd     uintptr
	callback uintptr
	deliver  func([]string)
}

func (target *fileDropTarget) handleEvent(evt event.Event, deliver func([]string), run func(func())) error {
	view, ok := evt.(app.Win32ViewEvent)
	if !ok {
		return nil
	}
	if !view.Valid() {
		run(target.close)
		return nil
	}
	if target.hwnd == view.HWND {
		return nil
	}
	var installErr error
	run(func() {
		target.close()
		installErr = target.install(view.HWND, deliver)
	})
	return installErr
}

func (target *fileDropTarget) install(hwnd uintptr, deliver func([]string)) error {
	target.hwnd = hwnd
	target.deliver = deliver
	target.callback = windows.NewCallback(target.windowProc)
	success, _, callErr := procSetSubclass.Call(target.hwnd, target.callback, dropSubclassID, 0)
	if success == 0 {
		target.hwnd = 0
		target.callback = 0
		target.deliver = nil
		return fmt.Errorf("enable file drop subclass: %w", callErr)
	}
	_, _, _ = procDragAcceptFiles.Call(target.hwnd, 1)
	return nil
}

func (target *fileDropTarget) close() {
	if target.hwnd == 0 {
		return
	}
	_, _, _ = procDragAcceptFiles.Call(target.hwnd, 0)
	_, _, _ = procRemoveSubclass.Call(target.hwnd, target.callback, dropSubclassID)
	target.hwnd = 0
	target.callback = 0
	target.deliver = nil
}

func (target *fileDropTarget) windowProc(hwnd uintptr, message uint32, wParam, lParam, _ uintptr, _ uintptr) uintptr {
	if message == wmDropFiles {
		paths := droppedPaths(wParam)
		_, _, _ = procDragFinish.Call(wParam)
		if len(paths) > 0 && target.deliver != nil {
			target.deliver(paths)
		}
		return 0
	}
	result, _, _ := procDefSubclass.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func droppedPaths(dropHandle uintptr) []string {
	count, _, _ := procDragQueryFileW.Call(dropHandle, ^uintptr(0), 0, 0)
	queryCount := count
	if queryCount > maxDroppedFiles+1 {
		queryCount = maxDroppedFiles + 1
	}
	paths := make([]string, 0, queryCount)
	for index := uintptr(0); index < queryCount; index++ {
		length, _, _ := procDragQueryFileW.Call(dropHandle, index, 0, 0)
		if length == 0 {
			continue
		}
		buffer := make([]uint16, length+1)
		// #nosec G103 -- DragQueryFileW requires a pointer to the writable UTF-16 buffer.
		written, _, _ := procDragQueryFileW.Call(dropHandle, index, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
		if written != length {
			continue
		}
		paths = append(paths, windows.UTF16ToString(buffer))
	}
	return paths
}
