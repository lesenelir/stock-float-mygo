package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo"

	"stock-float-mygo/internal/config"
	"stock-float-mygo/internal/l10n"
	"stock-float-mygo/internal/market"
)

// AlertsChanged sends the price alerts window what to show whenever quotes
// or settings change.
var AlertsChanged = mygo.NewEvent[AlertsState]("alerts")

// openAlerts opens the price alerts window, in front of the other apps.
// Its page reports its size, which shows it (Alerts.Fit).
func (a *App) openAlerts() {
	mygo.RunOnMain(func() {
		if a.alerts != nil {
			a.alerts.Destroy()
		}
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:             a.strings().AlertsTitle(),
			URL:               "/alerts.html",
			Width:             392,
			Height:            240,
			UseContentSize:    true,
			Hidden:            true,
			DisableResize:     true,
			DisableMinimize:   true,
			DisableMaximize:   true,
			DisableFullScreen: true,
			BackgroundColor:   "light-dark(#ececec, #282828)", // the page's --bg in src/settings.css
		})
		a.alerts = win
		win.OnClosed(func() {
			if a.alerts == win {
				a.alerts = nil
			}
		})
	})
}

// Alerts is what the price alerts window's page calls.
type Alerts struct{ app *App }

// AlertsState is the price alerts window's content: the symbols an alert
// can be set on, the alerts waiting to go off, and the texts around them.
type AlertsState struct {
	Strings AlertsStrings `json:"strings"`
	Symbols []AlertSymbol `json:"symbols"`
	Alerts  []AlertRow    `json:"alerts"`
}

// AlertSymbol is a symbol of the watch list, as the window offers it.
type AlertSymbol struct {
	ID string `json:"id"`
	// Text is the symbol as the settings write it, with its name where the
	// code is a number: 腾讯控股 hk00700.
	Text string `json:"text"`
	// Price is the newest price, or 0 before the first one.
	Price float64 `json:"price"`
	// Detail says the name and the newest price, to set the alert against.
	Detail string `json:"detail"`
	// PriceText is the newest price as the panel writes it, or "".
	PriceText string `json:"priceText"`
}

// AlertRow is an alert waiting to go off.
type AlertRow struct {
	Alert     config.Alert `json:"alert"`
	Label     string       `json:"label"`
	Condition string       `json:"condition"`
}

// AlertsStrings are the texts of the price alerts window.
type AlertsStrings struct {
	Title       string `json:"title"`
	Symbol      string `json:"symbol"`
	Price       string `json:"price"`
	Direction   string `json:"direction"`
	Rises       string `json:"rises"`
	Falls       string `json:"falls"`
	Add         string `json:"add"`
	Remove      string `json:"remove"`
	Done        string `json:"done"`
	NoAlerts    string `json:"noAlerts"`
	NeedSymbols string `json:"needSymbols"`
	Lang        string `json:"lang"`
}

// State returns what the window shows now; AlertsChanged sends the changes.
func (s Alerts) State() AlertsState { return alertsState(s.app.store.Snapshot(), s.app.strings()) }

func alertsState(snap Snapshot, s l10n.Strings) AlertsState {
	lang := "en"
	if s.Chinese() {
		lang = "zh"
	}
	state := AlertsState{
		Strings: AlertsStrings{
			Title: s.AlertsTitle(), Symbol: s.AlertSymbol(), Price: s.AlertPrice(),
			Direction: s.AlertDirection(), Rises: s.AlertRises(), Falls: s.AlertFalls(),
			Add: s.AddAlert(), Remove: s.RemoveAlert(), Done: s.Done(),
			NoAlerts: s.NoAlerts(), NeedSymbols: s.AlertsNeedSymbols(), Lang: lang,
		},
		Symbols: []AlertSymbol{},
		Alerts:  []AlertRow{},
	}
	for _, id := range snap.Config.Symbols {
		q, ok := snap.Quotes[id]
		sym := AlertSymbol{ID: id, Text: editableText(id)}
		var detail []string
		if ok && label(id, q, true) != sym.Text {
			// Names help where codes are numbers.
			sym.Text = label(id, q, true) + "  " + sym.Text
			detail = append(detail, q.Name)
		}
		if ok && q.Latest() > 0 {
			sym.Price, sym.PriceText = q.Latest(), priceText(q.Latest())
			detail = append(detail, s.LastPrice(sym.PriceText))
		}
		sym.Detail = strings.Join(detail, " · ")
		state.Symbols = append(state.Symbols, sym)
	}
	for _, alert := range snap.Config.Alerts {
		q, ok := snap.Quotes[alert.Symbol]
		state.Alerts = append(state.Alerts, AlertRow{
			Alert:     alert,
			Label:     label(alert.Symbol, q, ok),
			Condition: s.AlertCondition(alert.Below, priceText(alert.Price)),
		})
	}
	return state
}

// Add sets an alert on a symbol of the watch list. It fails, saying why,
// when the price is not one.
func (s Alerts) Add(symbol, price string, below bool) error {
	target, ok := parsePrice(price)
	if !ok {
		return errors.New(s.app.strings().InvalidPrice())
	}
	alert := config.Alert{Symbol: symbol, Price: target, Below: below}
	// Alerts touch nothing App.change brings in line: the store saves them
	// and sends the windows the change.
	s.app.store.Update(func(c *config.Config) {
		if slices.Contains(c.Symbols, symbol) && !slices.Contains(c.Alerts, alert) {
			c.Alerts = append(c.Alerts, alert)
		}
	})
	return nil
}

// Remove removes an alert.
func (s Alerts) Remove(alert config.Alert) {
	s.app.store.Update(func(c *config.Config) {
		if i := slices.Index(c.Alerts, alert); i >= 0 {
			c.Alerts = slices.Delete(c.Alerts, i, i+1)
		}
	})
}

// Fit sizes the price alerts window around its page, in DIPs, and shows it
// in front of the other apps.
func (s Alerts) Fit(ctx context.Context, width, height int) {
	Settings{s.app}.Fit(ctx, width, height)
}

// Close closes the price alerts window.
func (s Alerts) Close(ctx context.Context) {
	if win := mygo.CallerWindow(ctx); win != nil {
		win.Close()
	}
}

// parsePrice reads a price as typed, which must be above zero.
func parsePrice(text string) (float64, bool) {
	text = strings.ReplaceAll(strings.TrimSpace(text), ",", "")
	p, err := strconv.ParseFloat(text, 64)
	if err != nil || p <= 0 || math.IsInf(p, 0) || math.IsNaN(p) {
		return 0, false
	}
	return p, true
}

// priceText and percentText write numbers as the panel does (src/format.ts).
func priceText(price float64) string {
	if price < 1 {
		return fmt.Sprintf("%.3f", price)
	}
	return fmt.Sprintf("%.2f", price)
}

func percentText(percent float64) string { return fmt.Sprintf("%+.2f%%", percent) }

// notifyAlerts shows a system notification for each alert a quote set off.
// The store calls it from a feed's goroutine.
func (a *App) notifyAlerts(q market.Quote, hits []config.Alert) {
	s := a.strings()
	name := label(q.Symbol, q, true)
	var session string
	change := q.ChangePercent()
	if e := q.Extended; e != nil {
		session, change = s.Session(e.Session), e.ChangePercent()
	}
	body := s.AlertBody(session, priceText(q.Latest()), percentText(change))
	for _, hit := range hits {
		n := mygo.NewNotification(mygo.NotificationOptions{
			Title: s.AlertTitle(name, hit.Below, priceText(hit.Price)),
			Body:  body,
			Group: "price-alerts",
		})
		// Show waits for the system, and on the first notification for the
		// user to allow them.
		go func() {
			err := n.Show()
			switch {
			case err == nil:
			case errors.Is(err, mygo.ErrNotificationsDenied):
				// An alert the user set went unseen: say so, once a run.
				if a.notificationsDenied.CompareAndSwap(false, true) {
					mygo.Dialog.Error(s.NotificationsOff(), s.NotificationsOffDetail())
				}
			default:
				log.Printf("price alert %s: %v", n.ID(), err)
			}
		}()
	}
}
