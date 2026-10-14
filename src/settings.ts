// The settings window: the watch list and the data sources. Go checks and
// saves them (Settings.save), and says which symbols it could not read.
import { isCallError } from "mygo-runtime";
import { Settings } from "./mygo";

const $ = <T extends HTMLElement>(id: string) => document.getElementById(id) as T;

const form = $<HTMLFormElement>("settings");
const symbols = $<HTMLTextAreaElement>("symbols");
const longbridge = $<HTMLInputElement>("longbridge");
const finnhub = $<HTMLInputElement>("finnhub");
const error = $<HTMLParagraphElement>("error");

const { strings: s, ...saved } = await Settings.load();
document.documentElement.lang = s.lang;
document.title = s.title;
$("symbols-label").textContent = s.symbolsLabel;
$("symbols-hint").textContent = s.symbolsHint;
$("longbridge-label").textContent = s.longbridgeToggle;
$("longbridge-hint").textContent = s.longbridgeHint;
$("finnhub-label").textContent = s.finnhubKey;
finnhub.placeholder = s.finnhubPlaceholder;
$("cancel").textContent = s.cancel;
$("save").textContent = s.save;

symbols.value = saved.symbols;
longbridge.checked = saved.longbridge;
finnhub.value = saved.finnhubKey;
// Finnhub is used for US quotes only when Longbridge is off.
const syncFinnhub = () => (finnhub.disabled = longbridge.checked);
longbridge.addEventListener("change", syncFinnhub);
syncFinnhub();

form.addEventListener("submit", async (e) => {
  e.preventDefault();
  try {
    await Settings.save(symbols.value, longbridge.checked, finnhub.value);
  } catch (err) {
    if (!isCallError(err)) throw err;
    error.textContent = err.message;
    error.hidden = false;
    fit();
  }
});
$("cancel").addEventListener("click", () => void Settings.cancel());

addEventListener("keydown", (e) => {
  if (e.key === "Escape") {
    e.preventDefault();
    void Settings.cancel();
  } else if (e.key === "Enter" && !(e.target instanceof HTMLButtonElement) && (e.metaKey || e.target !== symbols)) {
    // Return saves, as the default button does; in the list it starts a
    // new line, and ⌘Return saves. A focused button does its own thing.
    e.preventDefault();
    form.requestSubmit();
  }
});

/** Size the window around the form; the first call shows it. */
function fit() {
  const { width, height } = form.getBoundingClientRect();
  void Settings.fit(Math.ceil(width), Math.ceil(height));
}

form.hidden = false;
fit();
symbols.focus();
