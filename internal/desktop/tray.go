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

	onReady := func() {
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
	}

	// systray.Register alone creates the tray window but never pumps its Win32
	// message loop, so WM_*BUTTONUP never reaches it and clicks do nothing.
	// RunWithExternalLoop's start() runs that pump in its own goroutine,
	// coexisting with Wails' own event loop.
	start, _ := systray.RunWithExternalLoop(onReady, func() {})
	start()
}
