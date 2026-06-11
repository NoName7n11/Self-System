package desktop

import (
	_ "embed"
	"log/slog"
	"runtime"

	"github.com/energye/systray"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed trayicon.ico
var trayIconICO []byte

//go:embed trayicon.png
var trayIconPNG []byte

// trayIcon returns the platform-appropriate icon bytes: Windows expects an
// .ico, other platforms a .png.
func trayIcon() []byte {
	if runtime.GOOS == "windows" {
		return trayIconICO
	}
	return trayIconPNG
}

// StartTray registers a system-tray icon with Show / Quit menu items. Wails v2
// has no native tray API, so this uses the energye/systray fork. Must be called
// after Startup has populated a.ctx.
func (a *App) StartTray() {
	if a.ctx == nil {
		slog.Warn("StartTray called before Startup; skipping tray")
		return
	}

	// The tray window, its message queue, Shell_NotifyIcon (icon add/modify),
	// and the message pump are all thread-affine on Windows: Win32 delivers a
	// window's messages only to the OS thread that created it. systray's
	// package init() locks only the main goroutine; Wails calls StartTray from
	// a different goroutine, and energye/systray's RunWithExternalLoop spawns an
	// unlocked goroutine for both onReady (so SetIcon races the wrong thread →
	// "unable to set icon: Unspecified error", missing tray icon) and for the
	// GetMessage pump (so window messages are never retrieved → dead clicks).
	//
	// Fix: do everything on ONE goroutine pinned to ONE OS thread — Register
	// (window creation, via RunWithExternalLoop with onReady=nil), then all the
	// icon/menu setup synchronously, then our own blocking message pump
	// (pumpTrayMessages) instead of systray's start(), which would pump on a
	// fresh unlocked goroutine.
	go func() {
		runtime.LockOSThread()

		// onReady=nil so Register runs synchronously here and does NOT spawn the
		// extra setup goroutine. start is systray's nativeStart (unlocked pump);
		// we discard it in favor of pumpTrayMessages on this thread.
		start, _ := systray.RunWithExternalLoop(nil, func() {})

		systray.SetIcon(trayIcon())
		systray.SetTitle("Self Systems")
		systray.SetTooltip("Self Systems — local-first knowledge & tasks")

		show := systray.AddMenuItem("Show", "Bring the window to the front")
		quit := systray.AddMenuItem("Quit", "Exit Self Systems")

		showWindow := func() {
			wruntime.WindowShow(a.ctx)
			wruntime.WindowUnminimise(a.ctx)
		}

		show.Click(showWindow)
		quit.Click(func() {
			wruntime.Quit(a.ctx)
		})

		// energye/systray does NOT show the menu by default — it only appears
		// when menu.ShowMenu() is called from a click handler. Without this,
		// right-clicking the tray icon does nothing. Left-click restores the
		// window directly.
		systray.SetOnRClick(func(menu systray.IMenu) { _ = menu.ShowMenu() })
		systray.SetOnClick(func(_ systray.IMenu) { showWindow() })

		// Blocking pump on this locked thread (Windows). On other platforms this
		// delegates to systray's own start().
		pumpTrayMessages(start)
	}()
}
