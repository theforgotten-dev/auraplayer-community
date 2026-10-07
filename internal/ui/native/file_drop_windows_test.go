//go:build windows

package native

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32DropTest = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalAlloc  = kernel32DropTest.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32DropTest.NewProc("GlobalLock")
	procGlobalUnlock = kernel32DropTest.NewProc("GlobalUnlock")
	procGlobalFree   = kernel32DropTest.NewProc("GlobalFree")
	procMoveMemory   = kernel32DropTest.NewProc("RtlMoveMemory")
)

func TestDroppedPathsReadsWideDropFilesPayload(t *testing.T) {
	want := []string{`C:\Music\first.mp3`, `D:\Audio\second track.MP3`}
	handle := newDropHandle(t, want)
	got := droppedPaths(handle)
	if len(got) != len(want) {
		t.Fatalf("droppedPaths() = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("droppedPaths()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func newDropHandle(t *testing.T, paths []string) uintptr {
	t.Helper()
	var names []uint16
	for _, path := range paths {
		names = append(names, utf16.Encode([]rune(path))...)
		names = append(names, 0)
	}
	names = append(names, 0)

	const dropFilesHeaderSize = 20
	totalBytes := dropFilesHeaderSize + len(names)*2
	payload := make([]byte, totalBytes)
	binary.LittleEndian.PutUint32(payload[0:4], dropFilesHeaderSize)
	binary.LittleEndian.PutUint32(payload[16:20], 1)
	for index, value := range names {
		binary.LittleEndian.PutUint16(payload[dropFilesHeaderSize+index*2:], value)
	}
	handle, _, callErr := procGlobalAlloc.Call(0x42, uintptr(totalBytes))
	if handle == 0 {
		t.Fatalf("GlobalAlloc: %v", callErr)
	}
	t.Cleanup(func() { procGlobalFree.Call(handle) })

	pointer, _, callErr := procGlobalLock.Call(handle)
	if pointer == 0 {
		t.Fatalf("GlobalLock: %v", callErr)
	}
	procMoveMemory.Call(pointer, uintptr(unsafe.Pointer(&payload[0])), uintptr(len(payload)))
	procGlobalUnlock.Call(handle)
	return handle
}
