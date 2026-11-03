// Package tray puts a Connect icon in the system tray / menu bar, with a
// small menu to open the web UI or quit the app. The browser window is
// just the UI: closing it doesn't stop Connect, only quitting from here
// (or Ctrl+C in a terminal) does.
package tray

import (
	_ "embed"
	"runtime"

	"github.com/getlantern/systray"
)

//go:embed icon.png
var iconPNG []byte

//go:embed icon.ico
var iconICO []byte

func icon() []byte {
	if runtime.GOOS == "windows" {
		return iconICO
	}
	return iconPNG
}

// Run blocks until Quit is called (or the OS asks the app to terminate),
// managing the tray icon's event loop. onOpen is called when the user picks
// "Open Connect" from the tray menu; onQuit is called once, when the user
// picks "Quit" or the tray is torn down for any other reason, so the caller
// can shut down the HTTP server and any background work.
func Run(onOpen func(), onQuit func()) {
	systray.Run(func() {
		systray.SetIcon(icon())
		systray.SetTitle("")
		systray.SetTooltip("Connect")

		open := systray.AddMenuItem("Open Connect", "Open the Connect web UI")
		systray.AddSeparator()
		quit := systray.AddMenuItem("Quit", "Stop Connect")

		go func() {
			for {
				select {
				case <-open.ClickedCh:
					onOpen()
				case <-quit.ClickedCh:
					systray.Quit()
					return
				}
			}
		}()
	}, onQuit)
}

// Quit tears down the tray icon, triggering the onQuit callback passed to
// Run. Safe to call even if the tray isn't running.
func Quit() {
	systray.Quit()
}
