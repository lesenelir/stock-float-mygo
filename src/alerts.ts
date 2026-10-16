// The price alerts window: the alerts waiting to go off, and a new one being
// set. Go keeps them, and sends what to show as quotes and alerts change
// (AlertsState); an alert goes away once it has notified.
import { isCallError } from "mygo-runtime";
import { Alerts, events, type AlertsState } from "./mygo";

const $ = <T extends HTMLElement>(id: string) => document.getElementById(id) as T;

const form = $<HTMLFormElement>("alerts");
const symbol = $<HTMLSelectElement>("symbol");
const price = $<HTMLInputElement>("price");
const rises = $<HTMLInputElement>("rises");
const falls = $<HTMLInputElement>("falls");
const error = $<HTMLParagraphElement>("error");
const list = $<HTMLUListElement>("list");

let state = await Alerts.state();
const s = state.strings;
document.documentElement.lang = s.lang;
document.title = s.title;
$("symbol-label").textContent = s.symbol;
$("price-label").textContent = s.price;
$("direction").setAttribute("aria-label", s.direction);
$("rises-label").textContent = s.rises;
$("falls-label").textContent = s.falls;
$("add").textContent = s.add;
$("done").textContent = s.done;
$("need-symbols").textContent = s.needSymbols;
$("empty").textContent = s.noAlerts;

// What each part was last built from: prices tick many times a second, and
// rebuilding the options or the list would drop a click in progress.
let builtOptions = "";
let builtList = "";

function render(next: AlertsState) {
  state = next;
  const watched = state.symbols.length > 0;
  $("new-symbol").hidden = $("new-price").hidden = !watched;
  $("need-symbols").hidden = watched;

  const options = state.symbols.map((sym) => `${sym.id}\t${sym.text}`).join("\n");
  if (options !== builtOptions) {
    builtOptions = options;
    const chosen = symbol.value;
    symbol.replaceChildren(...state.symbols.map((sym) => new Option(sym.text, sym.id)));
    if (state.symbols.some((sym) => sym.id === chosen)) symbol.value = chosen;
  }
  const current = state.symbols.find((sym) => sym.id === symbol.value);
  $("detail").textContent = current?.detail ?? "";
  price.placeholder = current?.priceText ?? "";

  const alerts = JSON.stringify(state.alerts.map((row) => [row.label, row.condition]));
  if (alerts !== builtList) {
    builtList = alerts;
    list.replaceChildren(
      ...state.alerts.map((row) => {
        const li = document.createElement("li");
        const label = span("label", row.label);
        const condition = span("condition", row.condition);
        const remove = document.createElement("button");
        remove.type = "button";
        remove.textContent = s.remove;
        remove.addEventListener("click", () => void Alerts.remove(row.alert));
        li.append(label, condition, remove);
        return li;
      }),
    );
  }
  $("empty").hidden = state.alerts.length > 0;
  fit();
}

function span(className: string, text: string) {
  const el = document.createElement("span");
  el.className = className;
  el.textContent = text;
  return el;
}

/** Picks rises or falls by which side of the last price the target is; the user can still change it. */
function guessDirection() {
  const target = Number(price.value.replaceAll(",", "").trim());
  const last = state.symbols.find((sym) => sym.id === symbol.value)?.price ?? 0;
  if (!(target > 0) || !(last > 0) || target === last) return;
  (target < last ? falls : rises).checked = true;
}

symbol.addEventListener("change", () => {
  render(state);
  guessDirection();
});
price.addEventListener("input", guessDirection);

form.addEventListener("submit", async (e) => {
  e.preventDefault();
  try {
    await Alerts.add(symbol.value, price.value, falls.checked);
    price.value = "";
    error.hidden = true;
  } catch (err) {
    if (!isCallError(err)) throw err;
    error.textContent = err.message;
    error.hidden = false;
  }
  fit();
  price.focus();
});
$("done").addEventListener("click", () => void Alerts.close());

addEventListener("keydown", (e) => {
  if (e.key === "Escape") {
    e.preventDefault();
    void Alerts.close();
  }
});

/** Size the window around the form; the first call shows it. */
function fit() {
  const { width, height } = form.getBoundingClientRect();
  void Alerts.fit(Math.ceil(width), Math.ceil(height));
}

events.alerts.on(render);
form.hidden = false;
render(state);
price.focus();
