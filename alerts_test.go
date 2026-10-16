package main

import (
	"path/filepath"
	"slices"
	"testing"

	"stock-float-mygo/internal/config"
	"stock-float-mygo/internal/feed"
	"stock-float-mygo/internal/l10n"
	"stock-float-mygo/internal/market"
)

func TestAlertsStateOffersTheWatchList(t *testing.T) {
	cfg := config.Default()
	cfg.Symbols = []string{"usAAPL", "hk00700"}
	cfg.Alerts = []config.Alert{{Symbol: "hk00700", Price: 400, Below: true}}
	snap := Snapshot{Config: cfg, Quotes: map[string]market.Quote{
		"usAAPL":  {Symbol: "usAAPL", Name: "苹果", Price: 190, PrevClose: 180},
		"hk00700": {Symbol: "hk00700", Name: "腾讯控股", Price: 421.2, PrevClose: 431},
	}}

	state := alertsState(snap, l10n.For(config.LanguageEn, ""))

	if got := state.Symbols[0]; got.Text != "AAPL" || got.Detail != "Last 190.00" || got.Price != 190 {
		t.Errorf("AAPL = %+v", got)
	}
	// Names help where codes are numbers.
	if got := state.Symbols[1]; got.Text != "腾讯控股  hk00700" || got.Detail != "腾讯控股 · Last 421.20" {
		t.Errorf("hk00700 = %+v", got)
	}
	if got := state.Alerts; len(got) != 1 || got[0].Label != "腾讯控股" || got[0].Condition != "Falls to 400.00" {
		t.Errorf("alerts = %+v", got)
	}
}

func TestAlertsAddsAndRemoves(t *testing.T) {
	a := &App{}
	a.store = NewStore(filepath.Join(t.TempDir(), "config.json"), nil)
	a.store.Update(func(c *config.Config) { c.Symbols = []string{"usAAPL"} })
	a.store.apply(a.store.generation, feed.Tencent, feed.QuoteEvent(market.Quote{Symbol: "usAAPL", Price: 190, PrevClose: 180}))
	alerts := Alerts{a}

	if err := alerts.Add("usAAPL", " 1,200.5 ", false); err != nil {
		t.Fatal(err)
	}
	// Not on the watch list: no feed would set it off.
	if err := alerts.Add("usNVDA", "100", false); err != nil {
		t.Fatal(err)
	}
	want := []config.Alert{{Symbol: "usAAPL", Price: 1200.5}}
	if got := a.store.Config().Alerts; !slices.Equal(got, want) {
		t.Errorf("alerts %+v, want %+v", got, want)
	}

	alerts.Remove(want[0])
	if got := a.store.Config().Alerts; len(got) != 0 {
		t.Errorf("alerts left %+v", got)
	}
}

// Add says why it refuses a price, in words of App.strings, which needs the
// running app: the reading of prices is tested alone.
func TestParsePriceTakesOnlyPricesAboveZero(t *testing.T) {
	for text, want := range map[string]float64{"200": 200, " 1,200.5 ": 1200.5, "0.051": 0.051} {
		if got, ok := parsePrice(text); !ok || got != want {
			t.Errorf("parsePrice(%q) = %v, %v", text, got, ok)
		}
	}
	for _, text := range []string{"", "abc", "0", "-3", "Inf", "NaN"} {
		if got, ok := parsePrice(text); ok {
			t.Errorf("parsePrice(%q) = %v", text, got)
		}
	}
}
