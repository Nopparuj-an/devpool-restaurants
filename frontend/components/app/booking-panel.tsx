"use client";

import { usePathname } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import { Button, ButtonLink } from "@/components/ui/button";
import { Segmented, Stepper } from "@/components/ui/controls";
import { Select } from "@/components/ui/field";
import { Badge, Notice } from "@/components/ui/misc";
import * as fmt from "@/lib/format";
import type { Availability, Reservation } from "@/lib/types";
import { useTimeZone } from "@/lib/use-time-zone";

const SLOT_MS = 15 * 60 * 1000;

export type BookingInput = { pax: number; starts_at: string; ends_at: string };

type Props = {
  seats: number;
  maxMinutes: number;
  cutoffMinutes: number;
  // Days the customer can pick, as ISO instants for the start of each day.
  days: { key: string; label: string }[];
  loadAvailability: (dayKey: string) => Promise<Availability>;
  onSubmit: (input: BookingInput) => Promise<{ error?: string }>;
  // When set, the panel edits this reservation instead of creating one.
  editing?: Reservation;
  // Day selected at first (defaults to the first of `days`).
  initialDay?: string;
  now: string;
  // Visitors can look at times but not book: the form is greyed out and points to login.
  locked?: boolean;
};

// Start times are offered only when every 15-minute slot the booking would
// cover has room for the party (the same rule the API enforces, R-BOOK-5).
export function BookingPanel({
  seats,
  maxMinutes,
  cutoffMinutes,
  days,
  loadAvailability,
  onSubmit,
  editing,
  initialDay,
  now,
  locked,
}: Props) {
  const tz = useTimeZone();
  const here = usePathname();
  const [dayKey, setDayKey] = useState(initialDay ?? days[0]?.key ?? "");
  const [reload, setReload] = useState(0);
  const [pax, setPax] = useState(editing?.pax ?? 2);
  const initialMinutes = editing ? (Date.parse(editing.ends_at) - Date.parse(editing.starts_at)) / 60000 : 60;
  const [minutes, setMinutes] = useState(Math.min(initialMinutes, maxMinutes));
  const [start, setStart] = useState<string | null>(editing?.starts_at ?? null);
  const [availability, setAvailability] = useState<Availability | null>(null);
  const [status, setStatus] = useState<{
    tone: "success" | "danger";
    text: string;
  } | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let live = true;
    loadAvailability(dayKey).then((a) => live && setAvailability(a));
    return () => {
      live = false;
    };
  }, [dayKey, loadAvailability, reload]);

  // Seats left per slot, giving back the seats of the booking being edited
  // so it doesn't block itself (R-EDIT-2).
  const slots = useMemo(() => {
    if (!availability) return [];
    return availability.slots.map((s) => {
      const t = Date.parse(s.start);
      const own = editing && t >= Date.parse(editing.starts_at) && t < Date.parse(editing.ends_at) ? editing.pax : 0;
      return { ...s, t, left: s.seats_left + own };
    });
  }, [availability, editing]);

  const needed = minutes / 15;
  const options = useMemo(() => {
    const nowMs = Date.parse(now);
    return slots
      .map((s, i) => {
        const run = slots.slice(i, i + needed);
        const contiguous = run.length === needed && run.every((r, j) => r.t === s.t + j * SLOT_MS);
        const minLeft = contiguous ? Math.min(...run.map((r) => r.left)) : 0;
        return { ...s, fits: contiguous && minLeft >= pax, minLeft };
      })
      .filter((s) => s.t > nowMs);
  }, [slots, needed, pax, now]);

  const chosen = options.find((o) => o.start === start && o.fits);
  const threshold = availability?.limited_threshold ?? 2;
  const durations = [];
  for (let m = 30; m <= maxMinutes; m += 30) durations.push(m);

  async function submit() {
    if (!chosen) return;
    setBusy(true);
    setStatus(null);
    const res = await onSubmit({
      pax,
      starts_at: chosen.start,
      ends_at: new Date(chosen.t + minutes * 60000).toISOString(),
    });
    setBusy(false);
    if (!res.error) {
      setStart(null);
      setReload((n) => n + 1); // show the seats this booking just took
    }
    setStatus(
      res.error
        ? { tone: "danger", text: res.error }
        : {
            tone: "success",
            text: editing ? "Your booking is updated." : "You're booked. See it under My bookings.",
          },
    );
  }

  return (
    <div className="flex flex-col gap-5 rounded-xl border border-line p-5">
      <div className="flex items-center justify-between gap-2">
        <h2 className="font-semibold">{editing ? "Change booking" : "Book a table"}</h2>
        {availability?.limited && <Badge tone="warning">Limited seats left</Badge>}
      </div>

      {locked && (
        <div className="flex flex-col gap-3 rounded-lg bg-accent-soft p-4 text-sm">
          <p className="font-medium">Log in to book a table.</p>
          <p className="text-muted">You can look at the times, but booking needs an account.</p>
          <div className="flex gap-2">
            <ButtonLink size="sm" href={`/login?next=${encodeURIComponent(here)}`}>
              Log in
            </ButtonLink>
            <ButtonLink size="sm" variant="secondary" href={`/signup?next=${encodeURIComponent(here)}`}>
              Sign up
            </ButtonLink>
          </div>
        </div>
      )}

      <fieldset disabled={locked} className={`m-0 flex min-w-0 flex-col gap-5 border-0 p-0 ${locked ? "opacity-50 saturate-0" : ""}`}>
        <div className="-mx-1 flex gap-1.5 overflow-x-auto px-1 pb-1">
          {days.map((d) => (
            <button
              key={d.key}
              type="button"
              onClick={() => {
                setDayKey(d.key);
                setStart(null);
              }}
              aria-pressed={d.key === dayKey}
              className="h-9 shrink-0 rounded-lg border border-line px-3 text-sm transition-colors hover:border-ink/30 aria-pressed:border-accent aria-pressed:bg-accent-soft aria-pressed:font-medium aria-pressed:text-accent"
            >
              {d.label}
            </button>
          ))}
        </div>

        <div className="flex flex-wrap items-end gap-4">
          <div className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">Guests</span>
            <Stepper value={pax} onChange={setPax} min={1} max={seats} label="Number of guests" />
          </div>
          <label className="flex min-w-32 flex-1 flex-col gap-1.5">
            <span className="text-sm font-medium">How long</span>
            <Select value={minutes} onChange={(e) => setMinutes(Number(e.target.value))}>
              {durations.map((m) => (
                <option key={m} value={m}>
                  {fmt.duration(m)}
                </option>
              ))}
            </Select>
          </label>
        </div>

        <div className="flex flex-col gap-2">
          <span className="text-sm font-medium">Time</span>
          {!availability ? (
            <div className="h-24 animate-pulse rounded-lg bg-surface" />
          ) : options.length === 0 ? (
            <p className="text-sm text-muted">Closed on this day.</p>
          ) : (
            <div className="-mr-2 grid max-h-64 grid-cols-4 gap-1.5 overflow-y-auto pr-2 sm:grid-cols-5">
              {options.map((o) => (
                <button
                  key={o.start}
                  type="button"
                  disabled={!o.fits}
                  onClick={() => setStart(o.start)}
                  aria-pressed={o.start === start && o.fits}
                  className="flex h-11 flex-col items-center justify-center rounded-lg border border-line text-sm tabular-nums transition-colors hover:border-ink/30 disabled:border-transparent disabled:bg-surface disabled:text-faint disabled:line-through aria-pressed:border-accent aria-pressed:bg-accent aria-pressed:text-white"
                >
                  {fmt.time(o.start, tz)}
                  {o.fits && o.minLeft <= threshold && <span className="text-[10px] leading-none opacity-70">{o.minLeft} left</span>}
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="flex flex-col gap-3 border-t border-line pt-4">
          <p className="text-sm">
            {chosen ? (
              <>
                <span className="font-medium">{fmt.day(chosen.start, tz)}</span>,{" "}
                {fmt.timeRange(chosen.start, new Date(chosen.t + minutes * 60000).toISOString(), tz)} · {fmt.guests(pax)}
              </>
            ) : (
              <span className="text-muted">Pick a time to continue.</span>
            )}
          </p>
          <Button onClick={submit} disabled={!chosen || busy}>
            {busy ? "Saving…" : editing ? "Save changes" : "Book table"}
          </Button>
          <p className="text-xs text-muted">You can change or cancel up to {fmt.duration(cutoffMinutes)} before your booking starts.</p>
          {status && <Notice tone={status.tone}>{status.text}</Notice>}
        </div>
      </fieldset>
    </div>
  );
}

// Segmented day filter reused by the owner table.
export function DayTabs({
  days,
  value,
  onChange,
}: {
  days: { key: string; label: string }[];
  value: string;
  onChange: (key: string) => void;
}) {
  return <Segmented label="Day" options={days.map((d) => ({ value: d.key, label: d.label }))} value={value} onChange={onChange} />;
}
