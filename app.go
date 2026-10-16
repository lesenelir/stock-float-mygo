package main

import (
	"log"
	"sync/atomic"

	"github.com/egoist/mygo"

	"stock-float-mygo/internal/config"
	"stock-float-mygo/internal/l10n"
)

// PanelChanged sends the panel what to show whenever quotes or settings
// change.
var PanelChanged = mygo.NewEvent[PanelState]("panel")

// App ties the store to the windows, menus and system integration.
//
// Fields marked "main thread" are only touched there: in event listeners,
// menu items and mygo.RunOnMain. Bound methods, which run on goroutines of
// their own, hop to it.
type App struct {
	store *Store

	panel    *mygo.Window // main thread, after WhenReady
	settings *mygo.Window // main thread; nil while closed
	alerts   *mygo.Window // main thread; nil while closed
	tray     *mygo.Tray   // main thread
	hotkey   string       // main thread; the registered accelerator, or ""

	placement        panelPlacement // main thread
	clickThroughStop chan struct{}  // main thread
	// quitting is set as a quit starts, when the panel may close. Main
	// thread.
	quitting bool

	updates updates
	// notificationsDenied is set once the user has been told that alerts
	// cannot notify them.
	notificationsDenied atomic.Bool
}

func newApp() *App {
	a := &App{}
	a.store = NewStore(config.Path(), a.pushState)
	a.store.OnAlert(a.notifyAlerts)
	return a
}

// strings says the interface in the configured language.
func (a *App) strings() l10n.Strings {
	return l10n.For(a.store.Config().Language, mygo.App.Locale())
}

// state is what the panel shows now.
func (a *App) state() PanelState {
	return panelState(a.store.Snapshot(), a.strings(), a.updates.offered())
}

// pushState sends the panel, and the alerts window while open, what they
// show now.
func (a *App) pushState() {
	var panel, alerts *mygo.Window
	mygo.RunOnMain(func() { panel, alerts = a.panel, a.alerts })
	if panel != nil {
		_ = PanelChanged.Emit(panel, a.state())
	}
	if alerts != nil {
		_ = AlertsChanged.Emit(alerts, alertsState(a.store.Snapshot(), a.strings()))
	}
}

// ready runs once the app has launched: it creates the panel and the menu
// bar icon, registers the shortcut, and starts the feeds and the updater.
func (a *App) ready() {
	cfg := a.store.Config()
	mygo.App.SetMenu(a.mainMenu())
	mygo.App.Dock.SetMenu(a.dockMenu())

	a.createPanel(cfg)
	a.applyClickThrough(cfg.ClickThrough)

	tray, err := mygo.NewTray(mygo.TrayOptions{Icon: trayIcon, IconIsTemplate: true, ToolTip: "StockFloat"})
	if err != nil {
		log.Println("menu bar icon:", err)
	} else {
		a.tray = tray
		// The menu is built as it opens, so its check marks are current.
		open := func() {
			tray.SetMenu(a.menu())
			tray.PopUpMenu()
			tray.SetMenu(nil)
		}
		tray.OnClick(open)
		tray.OnRightClick(open)
	}

	a.registerHotkey(cfg.Hotkey)
	a.store.Start()
	a.startUpdates()
}

// registerHotkey registers the shortcut that hides and shows the panel.
func (a *App) registerHotkey(spec string) {
	if a.hotkey != "" {
		mygo.GlobalShortcut.Unregister(a.hotkey)
		a.hotkey = ""
	}
	acc, ok := accelerator(spec)
	if !ok {
		if spec != "" {
			log.Printf("hotkey %q: not a shortcut, e.g. ctrl+opt+s", spec)
		}
		return
	}
	if err := mygo.GlobalShortcut.Register(acc, a.togglePanel); err != nil {
		log.Printf("hotkey %q: %v", spec, err)
		return
	}
	a.hotkey = acc
}

// change applies and saves a change of settings, then brings the rest of
// the app in line with it. The panel redraws itself from the store.
func (a *App) change(fn func(*config.Config)) {
	before, after := a.store.Update(fn)
	mygo.RunOnMain(func() {
		if after.Opacity != before.Opacity && a.panel != nil {
			a.panel.SetOpacity(after.Opacity)
		}
		if after.ClickThrough != before.ClickThrough {
			a.applyClickThrough(after.ClickThrough)
		}
		if after.DockIcon != before.DockIcon {
			mygo.App.SetActivationPolicy(activationPolicy(after))
		}
		if after.Language != before.Language {
			// Menus are built in a language; an open settings window too.
			mygo.App.SetMenu(a.mainMenu())
			mygo.App.Dock.SetMenu(a.dockMenu())
			if a.settings != nil {
				a.settings.Close()
			}
			if a.alerts != nil {
				a.alerts.Close()
			}
		}
	})
}

func activationPolicy(c config.Config) mygo.ActivationPolicy {
	if c.DockIcon {
		return mygo.ActivationPolicyRegular
	}
	return mygo.ActivationPolicyAccessory
}
