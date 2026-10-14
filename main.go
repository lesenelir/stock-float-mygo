// StockFloat shows live prices and daily change for the stocks you follow,
// in a small panel that floats above every app and never takes focus.
//
// Go does the work: it reads the settings, runs the quote feeds, and owns
// the menus, the menu bar icon, the shortcut and updates. The panel and the
// settings window are web pages (src/), which call Go through the client
// `mygo generate` writes to src/mygo.ts.
package main

import (
	_ "embed"
	"log"

	"github.com/egoist/mygo"
)

//go:embed assets/tray.png
var trayIcon []byte

func main() {
	// Opening the app again, from Finder or Spotlight, shows the panel of the
	// running one.
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	app := newApp()
	mygo.Bind(Panel{app})
	mygo.Bind(Settings{app})

	// The bundle is marked as a background app (LSUIElement in
	// mygo.config.ts), so that a user who turned the Dock icon off never sees
	// it flash; the Dock icon comes back here when it is on.
	mygo.App.SetActivationPolicy(activationPolicy(app.store.Config()))
	// The panel only ever hides, but the settings window closes: keep running
	// with no window open.
	mygo.App.OnWindowAllClosed(func() {})
	// Clicking the Dock icon, or opening the app again, brings a hidden
	// panel back.
	mygo.App.OnActivate(func(bool) { app.showPanel() })
	mygo.App.OnSecondInstance(func([]string, string) { app.showPanel() })
	mygo.App.OnQuit(app.store.Stop)

	mygo.App.WhenReady(app.ready)
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
