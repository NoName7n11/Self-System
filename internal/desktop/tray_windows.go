//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
)

// win32Point mirrors the Win32 POINT struct.
type win32Point struct{ X, Y int32 }

// win32Msg mirrors the Win32 MSG struct that GetMessageW fills in.
type win32Msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      win32Point
}

// pumpTrayMessages runs the Win32 message loop for the systray window on the
// CALLING goroutine, which StartTray has pinned to its OS thread. This replaces
// energye/systray's start()/nativeStart(), which pumps on a separate, unlocked
// goroutine — wrong, because Win32 delivers a window's messages only to the
// thread that created it, and Register created the systray window on this same
// locked thread. The start arg (systray's nativeStart) is intentionally unused.
func pumpTrayMessages(_ func()) {
	var msg win32Msg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		switch int32(ret) {
		case -1, 0:
			// -1 = error, 0 = WM_QUIT. Either way the loop ends.
			return
		default:
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}
