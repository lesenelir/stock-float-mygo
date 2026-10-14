package main

import (
	"slices"

	"stock-float-mygo/internal/config"
	"stock-float-mygo/internal/l10n"
	"stock-float-mygo/internal/market"
)

// PanelState is everything the panel shows, worded in the interface
// language. Go sends it whole on every change.
type PanelState struct {
	Rows []Row `json:"rows"`
	// EmptyHint is shown when the watch list is empty.
	EmptyHint string `json:"emptyHint,omitempty"`
	// Outages say which data sources are unreachable, and why.
	Outages []string `json:"outages"`
	// UpdateHint says a new version is out; quiet on purpose, as the panel
	// is on screen all day.
	UpdateHint string         `json:"updateHint,omitempty"`
	UpColor    config.UpColor `json:"upColor"`
	// Scale multiplies the fonts and the width of the panel.
	Scale float64 `json:"scale"`
}

// Row is one symbol of the watch list.
type Row struct {
	Symbol string `json:"symbol"`
	// Label is the ticker for US stocks, which are short already, and the
	// name for other markets, which read better by name.
	Label string `json:"label"`
	// Quote is null until the first price arrives.
	Quote *RowQuote `json:"quote"`
}

// RowQuote is the latest price of a row.
type RowQuote struct {
	Price         float64 `json:"price"`
	ChangePercent float64 `json:"changePercent"`
	// StaleAt is when the price, if it does not change, gets old enough to
	// be misleading, in milliseconds since the epoch; 0 when the source
	// does not say when it traded.
	StaleAt  int64        `json:"staleAt"`
	Extended *RowExtended `json:"extended,omitempty"`
}

// RowExtended is the price of a pre-market, post-market or overnight
// session, shown on a second line.
type RowExtended struct {
	Session market.Session `json:"session"`
	// Label names the session, for assistive technology.
	Label         string  `json:"label"`
	Price         float64 `json:"price"`
	ChangePercent float64 `json:"changePercent"`
}

// panelState builds what the panel shows from the store's state.
func panelState(snap Snapshot, s l10n.Strings, update string) PanelState {
	state := PanelState{
		Rows:    make([]Row, 0, len(snap.Config.Symbols)),
		Outages: []string{},
		UpColor: snap.Config.UpColor,
		Scale:   snap.Config.TextSize.Scale(),
	}
	for _, id := range snap.Config.Symbols {
		q, ok := snap.Quotes[id]
		row := Row{Symbol: id, Label: label(id, q, ok)}
		if ok {
			row.Quote = &RowQuote{Price: q.Price, ChangePercent: q.ChangePercent()}
			if !q.Time.IsZero() {
				row.Quote.StaleAt = q.Time.Add(market.StaleAfter).UnixMilli()
			}
			if e := q.Extended; e != nil {
				row.Quote.Extended = &RowExtended{
					Session: e.Session, Label: s.Session(e.Session),
					Price: e.Price, ChangePercent: e.ChangePercent(),
				}
			}
		}
		state.Rows = append(state.Rows, row)
	}
	if len(snap.Config.Symbols) == 0 {
		state.EmptyHint = s.EmptyHint()
	}
	// Two sources can be down for the same reason: say it once.
	for _, o := range snap.Outages {
		if msg := s.Outage(o); !slices.Contains(state.Outages, msg) {
			state.Outages = append(state.Outages, msg)
		}
	}
	slices.Sort(state.Outages)
	if update != "" {
		state.UpdateHint = s.UpdateHint(update)
	}
	return state
}

func label(id string, q market.Quote, ok bool) string {
	sym, valid := market.ParseID(id)
	switch {
	case !valid:
		return id
	case sym.Market == market.US:
		return sym.Code
	case ok && q.Name != "":
		return q.Name
	}
	return sym.Code
}
