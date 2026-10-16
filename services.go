package main

import (
	"context"
	"errors"
	"strings"
	"unicode"

	"github.com/egoist/mygo"

	"stock-float-mygo/internal/config"
)

// Panel is what the floating panel's page calls.
type Panel struct{ app *App }

// State returns what the panel shows now; PanelChanged sends the changes.
func (p Panel) State() PanelState { return p.app.state() }

// Resize fits the panel around its content, in DIPs. The first call places
// and shows the panel.
func (p Panel) Resize(width, height int) { p.app.resizePanel(width, height) }

// ShowMenu opens the panel's menu. On macOS the window opens it itself, as
// the page gets no clicks there.
func (p Panel) ShowMenu() { mygo.RunOnMain(p.app.showContextMenu) }

// Settings is what the settings window's page calls.
type Settings struct{ app *App }

// SettingsForm is the settings window's content: the saved settings and
// the texts around them.
type SettingsForm struct {
	Strings    SettingsStrings `json:"strings"`
	Symbols    string          `json:"symbols"`
	Longbridge bool            `json:"longbridge"`
	FinnhubKey string          `json:"finnhubKey"`
}

// SettingsStrings are the texts of the settings window.
type SettingsStrings struct {
	Title              string `json:"title"`
	SymbolsLabel       string `json:"symbolsLabel"`
	SymbolsHint        string `json:"symbolsHint"`
	LongbridgeToggle   string `json:"longbridgeToggle"`
	LongbridgeHint     string `json:"longbridgeHint"`
	FinnhubKey         string `json:"finnhubKey"`
	FinnhubPlaceholder string `json:"finnhubPlaceholder"`
	Cancel             string `json:"cancel"`
	Save               string `json:"save"`
	Lang               string `json:"lang"`
}

// Load returns the settings to edit.
func (s Settings) Load() SettingsForm {
	str := s.app.strings()
	cfg := s.app.store.Config()
	symbols := make([]string, len(cfg.Symbols))
	for i, id := range cfg.Symbols {
		symbols[i] = editableText(id)
	}
	lang := "en"
	if str.Chinese() {
		lang = "zh"
	}
	return SettingsForm{
		Strings: SettingsStrings{
			Title:              str.SettingsTitle(),
			SymbolsLabel:       str.SymbolsLabel(),
			SymbolsHint:        str.SymbolsHint(),
			LongbridgeToggle:   str.LongbridgeToggle(),
			LongbridgeHint:     str.LongbridgeHint(),
			FinnhubKey:         str.FinnhubKey(),
			FinnhubPlaceholder: str.FinnhubPlaceholder(),
			Cancel:             str.Cancel(),
			Save:               str.Save(),
			Lang:               lang,
		},
		Symbols:    strings.Join(symbols, "\n"),
		Longbridge: cfg.Longbridge,
		FinnhubKey: cfg.FinnhubKey,
	}
}

// Fit sizes the settings window around its page, in DIPs, and shows it in
// front of the other apps.
func (s Settings) Fit(ctx context.Context, width, height int) {
	win := mygo.CallerWindow(ctx)
	if win == nil {
		return
	}
	win.SetContentSize(width, height)
	if !win.IsVisible() {
		win.Center()
		mygo.App.Focus()
		win.Show()
	}
}

// Save checks and saves the settings, then closes the window. It fails,
// saying which entries, when some symbols cannot be read.
func (s Settings) Save(ctx context.Context, symbols string, longbridge bool, finnhubKey string) error {
	entries := strings.FieldsFunc(symbols, func(r rune) bool { return unicode.IsSpace(r) || r == ',' || r == '，' })
	ids, invalid := config.ParseSymbols(entries)
	if len(invalid) > 0 {
		return errors.New(s.app.strings().Unrecognized(invalid))
	}
	s.app.change(func(c *config.Config) {
		c.Symbols = ids
		c.Longbridge = longbridge
		c.FinnhubKey = strings.TrimSpace(finnhubKey)
		c.DropOrphanAlerts()
	})
	if win := mygo.CallerWindow(ctx); win != nil {
		win.Close()
	}
	return nil
}

// Cancel closes the settings window without saving.
func (s Settings) Cancel(ctx context.Context) {
	if win := mygo.CallerWindow(ctx); win != nil {
		win.Close()
	}
}
