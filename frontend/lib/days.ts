// Calendar days in a given IANA timezone, as UTC instants.

// Offset of timeZone from UTC at `at`, in milliseconds (e.g. +7h for Bangkok).
function offsetMs(at: number, timeZone: string): number {
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone,
    hourCycle: "h23",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).formatToParts(new Date(at));
  const get = (t: string) => Number(parts.find((p) => p.type === t)?.value);
  const asUTC = Date.UTC(get("year"), get("month") - 1, get("day"), get("hour"), get("minute"), get("second"));
  return asUTC - (at - (at % 1000));
}

// "2026-10-02" → the instant local midnight starts in timeZone.
export function startOfDay(key: string, timeZone: string): Date {
  const [y, m, d] = key.split("-").map(Number);
  const guess = Date.UTC(y, m - 1, d);
  let t = guess - offsetMs(guess, timeZone);
  t = guess - offsetMs(t, timeZone); // second pass settles DST edges
  return new Date(t);
}

export function dayKey(at: Date, timeZone: string): string {
  return new Intl.DateTimeFormat("en-CA", { timeZone, year: "numeric", month: "2-digit", day: "2-digit" }).format(at);
}

export function addDays(key: string, n: number): string {
  const [y, m, d] = key.split("-").map(Number);
  return new Date(Date.UTC(y, m - 1, d + n)).toISOString().slice(0, 10);
}

// [from, to) covering one local day.
export function dayRange(key: string, timeZone: string): { from: string; to: string } {
  return {
    from: startOfDay(key, timeZone).toISOString(),
    to: startOfDay(addDays(key, 1), timeZone).toISOString(),
  };
}

// The next `count` days starting today, labelled "Today", "Tomorrow", "Sat 3".
export function upcomingDays(now: Date, timeZone: string, count: number): { key: string; label: string }[] {
  const today = dayKey(now, timeZone);
  return Array.from({ length: count }, (_, i) => {
    const key = addDays(today, i);
    const label =
      i === 0
        ? "Today"
        : i === 1
          ? "Tomorrow"
          : new Intl.DateTimeFormat("en-GB", { weekday: "short", day: "numeric", timeZone: "UTC" }).format(new Date(`${key}T12:00:00Z`));
    return { key, label };
  });
}
