//go:build !windows

package desktop

// pumpTrayMessages delegates to systray's own start() on non-Windows platforms,
// where energye/systray drives its GTK loop and the Windows GetMessage pump
// does not apply.
func pumpTrayMessages(start func()) {
	start()
}
