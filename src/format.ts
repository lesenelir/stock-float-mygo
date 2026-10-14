import type { UpColor } from "./mygo";

/** A price, with three decimals below 1, where cents are too coarse. */
export function priceText(price: number): string {
  return price.toFixed(price < 1 ? 3 : 2);
}

/** A change in percent, always signed: +0.89%, -2.27%. */
export function percentText(percent: number): string {
  return `${percent >= 0 ? "+" : ""}${percent.toFixed(2)}%`;
}

/** The color class of a gain or loss: "up", "down", "flat", or "" in the no-color scheme. */
export function tint(change: number, upColor: UpColor): string {
  if (upColor === "mono") return "";
  if (change === 0) return "flat";
  return change > 0 === (upColor === "green") ? "green" : "red";
}
