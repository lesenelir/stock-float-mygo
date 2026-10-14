package main

import (
	"github.com/egoist/mygo"

	"stock-float-mygo/internal/market"
)

// openSettings opens the settings window, in front of the other apps. Its
// page reports its size, which shows it (Settings.Fit).
func (a *App) openSettings() {
	mygo.RunOnMain(func() {
		// Opened again, it starts over from the saved settings.
		if a.settings != nil {
			a.settings.Destroy()
		}
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:             a.strings().SettingsTitle(),
			URL:               "/settings.html",
			Width:             392,
			Height:            420,
			UseContentSize:    true,
			Hidden:            true,
			DisableResize:     true,
			DisableMinimize:   true,
			DisableMaximize:   true,
			DisableFullScreen: true,
			BackgroundColor:   "light-dark(#ececec, #282828)", // the page's --bg in src/settings.css
		})
		a.settings = win
		win.OnClosed(func() {
			if a.settings == win {
				a.settings = nil
			}
		})
	})
}

// editableText is a symbol as the settings take it: US symbols without
// their prefix, as the hint asks for them.
func editableText(id string) string {
	if s, ok := market.ParseID(id); ok && s.Market == market.US {
		return s.Code
	}
	return id
}
