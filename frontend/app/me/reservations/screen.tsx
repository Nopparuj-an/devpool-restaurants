"use client";

import { useState } from "react";

import { ReservationCard } from "@/components/app/reservation-card";
import { Page, PageTitle, SiteHeader } from "@/components/app/site-header";
import { ButtonLink } from "@/components/ui/button";
import { ConfirmDialog, Segmented } from "@/components/ui/controls";
import { EmptyState } from "@/components/ui/misc";
import { LoadMore } from "@/components/ui/pagination";
import * as fmt from "@/lib/format";
import type { Account, Reservation } from "@/lib/types";
import { useTimeZone } from "@/lib/use-time-zone";

export function MyBookingsScreen({
  account,
  reservations,
  total,
  onMore,
  onCancel,
  onChange,
}: {
  account: Account;
  reservations: Reservation[];
  // All bookings; `reservations` may be the first pages only (current ones come first).
  total?: number;
  onMore?: () => Promise<void>;
  onCancel: (r: Reservation) => Promise<void>;
  onChange: (r: Reservation) => void;
}) {
  const [tab, setTab] = useState<"upcoming" | "past">("upcoming");
  const [cancelling, setCancelling] = useState<Reservation | null>(null);
  const tz = useTimeZone();
  const upcoming = reservations.filter((r) => r.state === "upcoming" || r.state === "in_progress");
  const past = reservations.filter((r) => r.state === "completed" || r.state === "cancelled");
  const list = tab === "upcoming" ? upcoming : past;
  return (
    <>
      <SiteHeader account={account} current="/me/reservations" />
      <Page>
        <PageTitle title="My bookings" />
        <div className="mb-6">
          <Segmented
            label="Show"
            value={tab}
            onChange={setTab}
            options={[
              { value: "upcoming", label: `Upcoming (${upcoming.length})` },
              { value: "past", label: "Past" },
            ]}
          />
        </div>
        {list.length === 0 ? (
          <EmptyState
            title={tab === "upcoming" ? "No upcoming bookings" : "No past bookings yet"}
            action={tab === "upcoming" && <ButtonLink href="/">Find a table</ButtonLink>}
          />
        ) : (
          <div className="flex max-w-2xl flex-col gap-3">
            {list.map((r) => (
              <ReservationCard key={r.id} reservation={r} onCancel={() => setCancelling(r)} onChange={() => onChange(r)} />
            ))}
          </div>
        )}
        {tab === "past" && onMore && <LoadMore shown={reservations.length} total={total ?? reservations.length} onMore={onMore} />}
        <ConfirmDialog
          open={cancelling !== null}
          title="Cancel this booking?"
          confirmLabel="Cancel booking"
          danger
          onClose={() => setCancelling(null)}
          onConfirm={async () => {
            if (cancelling) await onCancel(cancelling);
            setCancelling(null);
          }}
        >
          {cancelling && (
            <>
              {cancelling.restaurant.name}, {fmt.day(cancelling.starts_at, tz)} at {fmt.time(cancelling.starts_at, tz)} for{" "}
              {fmt.guests(cancelling.pax)}. The seats go back to the restaurant.
            </>
          )}
        </ConfirmDialog>
      </Page>
    </>
  );
}
