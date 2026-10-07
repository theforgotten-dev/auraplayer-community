//go:build windows

package native

import (
	"fmt"
	"os"
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/event"
	"golang.org/x/sys/windows"
)

const (
	wmSetIcon  = 0x0080
	iconSmall  = 0
	iconBig    = 1
	iconSmall2 = 2
)

var (
	shell32Icon       = windows.NewLazySystemDLL("shell32.dll")
	user32Icon        = windows.NewLazySystemDLL("user32.dll")
	procExtractIconEx = shell32Icon.NewProc("ExtractIconExW")
	procSendMessage   = user32Icon.NewProc("SendMessageW")
	procDestroyIcon   = user32Icon.NewProc("DestroyIcon")
)

// taskbarIcon installs the executable's embedded icon on Gio's native window.
// Windows then uses the same identity in the title switcher and taskbar.
type taskbarIcon struct {
	hwnd  uintptr
	large uintptr
	small uintptr
}

func (icon *taskbarIcon) handleEvent(evt event.Event, run func(func())) error {
	view, ok := evt.(app.Win32ViewEvent)
	if !ok {
		return nil
	}
	if !view.Valid() {
		run(icon.close)
		return nil
	}
	if icon.hwnd == view.HWND {
		return nil
	}

	var installErr error
	run(func() {
		icon.close()
		installErr = icon.install(view.HWND)
	})
	return installErr
}

func (icon *taskbarIcon) install(hwnd uintptr) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	executableUTF16, err := windows.UTF16PtrFromString(executable)
	if err != nil {
		return fmt.Errorf("encode executable path: %w", err)
	}

	var large, small uintptr
	// #nosec G103 -- ExtractIconExW writes the two HICON handles supplied here.
	count, _, callErr := procExtractIconEx.Call(
		uintptr(unsafe.Pointer(executableUTF16)),
		0,
		uintptr(unsafe.Pointer(&large)),
		uintptr(unsafe.Pointer(&small)),
		1,
	)
	if count == 0 || (large == 0 && small == 0) {
		return fmt.Errorf("extract embedded icon: %w", callErr)
	}
	if large == 0 {
		large = small
	}
	if small == 0 {
		small = large
	}

	icon.hwnd = hwnd
	icon.large = large
	icon.small = small
	_, _, _ = procSendMessage.Call(hwnd, wmSetIcon, iconBig, large)
	_, _, _ = procSendMessage.Call(hwnd, wmSetIcon, iconSmall, small)
	_, _, _ = procSendMessage.Call(hwnd, wmSetIcon, iconSmall2, small)
	return nil
}

func (icon *taskbarIcon) close() {
	if icon.hwnd != 0 {
		_, _, _ = procSendMessage.Call(icon.hwnd, wmSetIcon, iconBig, 0)
		_, _, _ = procSendMessage.Call(icon.hwnd, wmSetIcon, iconSmall, 0)
		_, _, _ = procSendMessage.Call(icon.hwnd, wmSetIcon, iconSmall2, 0)
	}
	if icon.large != 0 {
		_, _, _ = procDestroyIcon.Call(icon.large)
	}
	if icon.small != 0 && icon.small != icon.large {
		_, _, _ = procDestroyIcon.Call(icon.small)
	}
	icon.hwnd = 0
	icon.large = 0
	icon.small = 0
}
