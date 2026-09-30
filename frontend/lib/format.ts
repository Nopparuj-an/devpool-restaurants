// Display formatting. Times are shown in the viewer's timezone (R-TIME-2);
// components get the zone from useTimeZone() so server and client agree.

export const WEEKDAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

export function rating(r: number | null): string {
  return r === null ? "" : r.toFixed(1);
}

export function reviewCount(n: number): string {
  return n === 1 ? "1 review" : `${n} reviews`;
}

// "1 restaurant", "1,204 reviews".
export function count(n: number, noun: string): string {
  return n === 1 ? `1 ${noun}` : `${n.toLocaleString("en")} ${noun}s`;
}

export function guests(n: number): string {
  return n === 1 ? "1 guest" : `${n} guests`;
}

export function time(iso: string, timeZone: string): string {
  return new Intl.DateTimeFormat("en-GB", { hour: "2-digit", minute: "2-digit", timeZone }).format(new Date(iso));
}

export function day(iso: string, timeZone: string): string {
  return new Intl.DateTimeFormat("en-GB", { weekday: "short", day: "numeric", month: "short", timeZone }).format(new Date(iso));
}

export function timeRange(start: string, end: string, timeZone: string): string {
  return `${time(start, timeZone)} to ${time(end, timeZone)}`;
}

export function duration(minutes: number): string {
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  if (h === 0) return `${m} min`;
  return m === 0 ? `${h} h` : `${h} h ${m} min`;
}

// "18:00 to 02:00" with a note when the shift ends the next day or runs 24 hours (R-HOURS-3).
export function shift(open: string, close: string): string {
  if (open === close) return "Open 24 hours";
  return close < open ? `${open} to ${close} (next day)` : `${open} to ${close}`;
}
