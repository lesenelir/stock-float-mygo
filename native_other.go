//go:build !darwin

package main

import "github.com/egoist/mygo"

// makePanel keeps the panel on every workspace. On Linux and Windows the
// page drags the window (--app-region: drag) and opens its menu on
// right-click itself; MyGo has no way to keep it from taking the keyboard
// there.
func makePanel(win *mygo.Window, onRightClick func()) {
	win.SetVisibleOnAllWorkspaces(true)
}

func refreshShadow(*mygo.Window) {}

// optionKeyDown has no portable equivalent: click-through stays on.
func optionKeyDown() bool { return false }
