"use client";

import Link from "next/link";

import { Button } from "@/components/ui/button";
import { Badge, Photo } from "@/components/ui/misc";
import * as fmt from "@/lib/format";
import type { Reservation, ReservationState } from "@/lib/types";
import { useTimeZone } from "@/lib/use-time-zone";

const stateBadge: Record<ReservationState, { tone: "accent" | "success" | "neutral" | "danger"; label: string }> = {
  upcoming: { tone: "accent", label: "Upcoming" },
  in_progress: { tone: "success", label: "Now" },
  completed: { tone: "neutral", label: "Visited" },
  cancelled: { tone: "neutral", label: "Cancelled" },
};

export function ReservationCard({
  reservation: r,
  onChange,
  onCancel,
}: {
  reservation: Reservation;
  onChange?: () => void;
  onCancel?: () => void;
}) {
  const tz = useTimeZone();
  const badge = stateBadge[r.state];
  return (
    <div className={`flex gap-4 rounded-xl border border-line p-4 ${r.state === "cancelled" ? "opacity-60" : ""}`}>
      <Photo src={r.restaurant.cover_url} className="size-20 shrink-0 rounded-lg sm:size-24" />
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex items-start justify-between gap-2">
          <Link href={`/restaurants/${r.restaurant.id}`} className="truncate font-medium hover:text-accent">
            {r.restaurant.name}
          </Link>
          <Badge tone={badge.tone}>{badge.label}</Badge>
        </div>
        <p className="text-sm">
          {fmt.day(r.starts_at, tz)}, {fmt.timeRange(r.starts_at, r.ends_at, tz)}
        </p>
        <p className="text-sm text-muted">{fmt.guests(r.pax)}</p>
        {r.state === "upcoming" && (
          <div className="mt-2 flex flex-wrap items-center gap-2">
            {r.can_modify ? (
              <>
                <Button size="sm" variant="secondary" onClick={onChange}>
                  Change
                </Button>
                <Button size="sm" variant="ghost" onClick={onCancel}>
                  Cancel booking
                </Button>
                <span className="text-xs text-muted">
                  Until {fmt.time(r.modifiable_until, tz)} on {fmt.day(r.modifiable_until, tz)}
                </span>
              </>
            ) : (
              <span className="text-xs text-muted">Too late to change or cancel. The cutoff was {fmt.time(r.modifiable_until, tz)}.</span>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
