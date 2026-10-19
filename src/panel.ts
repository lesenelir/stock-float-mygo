// The floating panel: one line per symbol with its price and daily change,
// a second line outside the regular session, and what is wrong when a data
// source is down. Go sends everything to show (PanelState); this page only
// lays it out, and reports its size so the window can fit around it.
import { isMyGo, runtime } from "mygo-runtime";
import { Panel, events, type PanelState, type Row } from "./mygo";
import { percentText, priceText, tint } from "./format";

const panel = document.querySelector<HTMLElement>("#panel")!;
const rowsEl = div("rows");
const notesEl = div("notes");
panel.append(rowsEl, notesEl);

/** Each symbol's row, kept from one update to the next so a flash plays to its end. */
let rowEls = new Map<string, HTMLElement>();
/** The last regular price of each symbol, to flash its row when it moves. */
const lastPrices = new Map<string, number>();

function render(state: PanelState) {
  document.documentElement.style.setProperty("--scale", String(state.scale));

  const els = state.rows.map((row) => {
    const el = rowEls.get(row.symbol) ?? div("row");
    fillRow(el, row, state);
    return el;
  });
  const order = state.rows.map((r) => r.symbol).join();
  if (order !== [...rowEls.keys()].join()) rowsEl.replaceChildren(...els);
  rowEls = new Map(state.rows.map((row, i) => [row.symbol, els[i]!]));

  const notes: HTMLElement[] = [];
  if (state.emptyHint) notes.push(div("note empty", state.emptyHint));
  for (const outage of state.outages) notes.push(div("note outage", outage));
  if (state.updateHint) notes.push(div("note update", state.updateHint));
  notesEl.replaceChildren(...notes);

  for (const [i, row] of state.rows.entries()) {
    const price = row.quote?.price;
    const old = lastPrices.get(row.symbol);
    if (price === undefined) {
      lastPrices.delete(row.symbol);
      continue;
    }
    lastPrices.set(row.symbol, price);
    // Only regular-session moves flash, and only while the user wants them
    // to; extended-hours ticks update quietly.
    if (state.flash && old !== undefined && old !== price && !row.quote?.extended) {
      flash(els[i]!, tint(price - old, state.upColor) || "gray");
    }
  }
  fadeStale();
}

function fillRow(el: HTMLElement, row: Row, state: PanelState) {
  const q = row.quote;
  if (q?.staleAt) el.dataset.staleAt = String(q.staleAt);
  else delete el.dataset.staleAt;

  // The name sits on the first line; the extended-hours line hangs under
  // the price column only.
  const prices = div("prices");
  const main = div("line");
  main.append(
    div("price", q ? priceText(q.price) : "—"),
    div(`change ${q ? tint(q.changePercent, state.upColor) : ""}`, q ? percentText(q.changePercent) : "—"),
  );
  prices.append(main);

  const e = q?.extended;
  if (e) {
    const line = div("line extended");
    // The icon hugs the price it labels: a label, not a third number.
    const price = div("price");
    const icon = div(`session ${e.session}`);
    icon.setAttribute("role", "img");
    icon.setAttribute("aria-label", e.label);
    price.append(icon, priceText(e.price));
    line.append(price, div(`change ${tint(e.changePercent, state.upColor)}`, percentText(e.changePercent)));
    prices.append(line);
  }
  el.replaceChildren(div("name", row.label), prices);
}

function div(className: string, text?: string): HTMLElement {
  const el = document.createElement("div");
  el.className = className;
  if (text !== undefined) el.textContent = text;
  return el;
}

/** Light the row up in the color of the move, then fade it out. */
function flash(row: HTMLElement, color: string) {
  const c = getComputedStyle(row).getPropertyValue(`--${color}`);
  const lit = `color-mix(in srgb, ${c} 30%, transparent)`;
  row.animate(
    [
      { backgroundColor: "transparent" },
      { backgroundColor: lit, offset: 0.14, easing: "ease-out" },
      { backgroundColor: "transparent" },
    ],
    { duration: 700, easing: "ease-out" },
  );
}

/** Dim the rows whose price has not updated for half an hour: a closed market, a lunch break or a lost connection. */
function fadeStale() {
  const now = Date.now();
  for (const row of rowEls.values()) {
    const staleAt = Number(row.dataset.staleAt ?? 0);
    row.classList.toggle("stale", staleAt > 0 && now > staleAt);
  }
}

// A price that stops updating fades without new data arriving.
setInterval(fadeStale, 60_000);

// The window fits around the page, keeping its top-left corner in place.
new ResizeObserver(() => {
  const { width, height } = panel.getBoundingClientRect();
  void Panel.resize(Math.ceil(width), Math.ceil(height));
}).observe(panel);

// On macOS the window takes the mouse itself: drag anywhere to move the
// panel, right-click for its menu. Elsewhere the page does both.
if (isMyGo() && runtime().platform !== "darwin") {
  document.documentElement.classList.add("drag");
  addEventListener("contextmenu", (e) => {
    e.preventDefault();
    void Panel.showMenu();
  });
}

events.panel.on(render);
render(await Panel.state());
