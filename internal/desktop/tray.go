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

		show.Click(func() {
			wruntime.WindowShow(a.ctx)
			wruntime.WindowUnminimise(a.ctx)
		})
		quit.Click(func() {
			wruntime.Quit(a.ctx)
		})
	}

	systray.Register(onReady, func() {})
}
