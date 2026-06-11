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

// StartTray registers a system-tray icon with Show / Quit menu items. It uses
// systray.Register (non-blocking) rather than systray.Run so it coexists with
// the Wails main event loop instead of taking it over. Wails v2 has no native
// tray API, so this uses the energye/systray fork. Must be called after
// Startup has populated a.ctx.
func (a *App) StartTray() {
	if a.ctx == nil {
		slog.Warn("StartTray called before Startup; skipping tray")
		return
	}

	// systray.Register alone creates the tray window but never pumps its Win32
	// message loop, so WM_*BUTTONUP never reaches it and clicks do nothing.
	// RunWithExternalLoop's start() runs that pump.
	//
	// Win32 windows, message queues, and Shell_NotifyIcon (NIM_ADD/NIM_MODIFY)
	// are thread-affine: they must all be called from the SAME OS thread that
	// created the tray window. systray's package init() calls
	// runtime.LockOSThread(), which only pins whichever goroutine ran init (the
	// main goroutine) — but Wails calls OnStartup (and so StartTray) from a
	// different goroutine, and RunWithExternalLoop's onReady callback runs on
	// yet another, unlocked goroutine spawned internally by Register(). Calling
	// SetIcon/SetTooltip/AddMenuItem from that onReady goroutine hits a
	// different OS thread than the one that created the window, so
	// Shell_NotifyIcon's NIM_MODIFY fails ("systray error: unable to set icon:
	// Unspecified error") and the icon never appears; separately, GetMessage in
	// start() can also end up pumping the wrong thread's queue, so clicks do
	// nothing.
	//
	// Fix: run Register (via RunWithExternalLoop with onReady=nil, so it
	// doesn't spawn that extra goroutine) and all the icon/menu setup calls
	// synchronously, in a single goroutine locked to one OS thread, before
	// entering the blocking message pump on that same thread.
	go func() {
		runtime.LockOSThread()

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
		// when menu.ShowMenu() is called from a click handler (see the package
		// note in systray.go). Without this, right-clicking the tray icon does
		// nothing. Left-click restores the window directly.
		systray.SetOnRClick(func(menu systray.IMenu) { _ = menu.ShowMenu() })
		systray.SetOnClick(func(_ systray.IMenu) { showWindow() })

		start()
	}()
}
