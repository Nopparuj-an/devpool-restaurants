"use client";

import Link from "next/link";

import { Badge } from "@/components/ui/misc";
import * as fmt from "@/lib/format";
import type { Reservation } from "@/lib/types";
import { useTimeZone } from "@/lib/use-time-zone";

const SLOT_MS = 15 * 60 * 1000;

// Guests in the room per 15-minute slot, as a bar chart over opening hours.
export function LoadStrip({ reservations, seats, from, to }: { reservations: Reservation[]; seats: number; from: string; to: string }) {
  const tz = useTimeZone();
  const start = Date.parse(from);
  const end = Date.parse(to);
  const active = reservations.filter((r) => r.status === "active");
  const bars = [];
  for (let t = start; t < end; t += SLOT_MS) {
    const load = active.filter((r) => Date.parse(r.starts_at) <= t && Date.parse(r.ends_at) > t).reduce((sum, r) => sum + r.pax, 0);
    bars.push({ t, load });
  }
  const peak = Math.max(0, ...bars.map((b) => b.load));
  const hours = bars.filter((b) => new Date(b.t).getUTCMinutes() === 0);
  return (
    <figure className="flex flex-col gap-2">
      <figcaption className="flex justify-between text-sm">
        <span className="font-medium">Guests in the room</span>
        <span className="text-muted">
          Peak {peak} of {seats} seats
        </span>
      </figcaption>
      <div className="flex h-20 items-end gap-px rounded-lg bg-surface px-1 pt-1">
        {bars.map((b) => (
          <div
            key={b.t}
            title={`${fmt.time(new Date(b.t).toISOString(), tz)}: ${b.load} of ${seats}`}
            className={`flex-1 rounded-t-sm ${b.load >= seats ? "bg-accent" : b.load > 0 ? "bg-accent/40" : "bg-transparent"}`}
            style={{ height: `${Math.max(b.load ? 6 : 0, (b.load / seats) * 100)}%` }}
          />
        ))}
      </div>
      <div className="flex justify-between text-xs text-faint tabular-nums">
        {hours
          .filter((_, i) => i % 2 === 0)
          .map((b) => (
            <span key={b.t}>{fmt.time(new Date(b.t).toISOString(), tz)}</span>
          ))}
      </div>
    </figure>
  );
}

// Everyone booked for the day, with contact details for the owner (R-PRIV-2).
export function OwnerTable({ reservations }: { reservations: Reservation[] }) {
  const tz = useTimeZone();
  return (
    <div className="overflow-x-auto rounded-xl border border-line">
      <table className="w-full min-w-[36rem] text-left text-sm">
        <thead className="border-b border-line bg-surface text-muted">
          <tr>
            <th className="px-4 py-2.5 font-medium">Time</th>
            <th className="px-4 py-2.5 font-medium">Guests</th>
            <th className="px-4 py-2.5 font-medium">Name</th>
            <th className="px-4 py-2.5 font-medium">Email</th>
            <th className="px-4 py-2.5 font-medium">Status</th>
          </tr>
        </thead>
        <tbody>
          {reservations.map((r) => (
            <tr key={r.id} className="border-b border-line last:border-0">
              <td className="px-4 py-3 tabular-nums">{fmt.timeRange(r.starts_at, r.ends_at, tz)}</td>
              <td className="px-4 py-3 tabular-nums">{r.pax}</td>
              <td className="px-4 py-3 font-medium">
                {r.customer && (
                  <Link href={`/users/${r.customer.id}`} className="hover:text-accent">
                    {r.customer.display_name}
                  </Link>
                )}
              </td>
              <td className="px-4 py-3 text-muted">{r.customer?.email}</td>
              <td className="px-4 py-3">{r.status === "cancelled" ? <Badge>Cancelled</Badge> : <Badge tone="accent">Booked</Badge>}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
