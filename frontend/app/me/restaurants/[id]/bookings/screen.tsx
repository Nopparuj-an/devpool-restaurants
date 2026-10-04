"use client";

import { DayTabs } from "@/components/app/booking-panel";
import { LoadStrip, OwnerTable } from "@/components/app/owner-table";
import { Page, PageTitle, SiteHeader } from "@/components/app/site-header";
import { ButtonLink } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/misc";
import type { Account, Day, Reservation, RestaurantDetail } from "@/lib/types";

export function OwnerBookingsScreen({
  account,
  restaurant,
  days,
  day,
  onDay,
  reservations,
  openWindow,
  loading,
}: {
  account: Account;
  restaurant: RestaurantDetail;
  days: Day[];
  day: string;
  onDay: (day: string) => void;
  reservations: Reservation[];
  // Opening hours of the selected day, for the load chart.
  openWindow: { from: string; to: string } | null;
  loading?: boolean;
}) {
  const booked = reservations.filter((r) => r.status === "active");
  return (
    <>
      <SiteHeader account={account} current="/me/restaurants" />
      <Page>
        <PageTitle
          title="Bookings"
          subtitle={restaurant.name}
          action={
            <ButtonLink variant="secondary" href={`/me/restaurants/${restaurant.id}/edit`}>
              Edit restaurant
            </ButtonLink>
          }
        />
        <div className="mb-6 overflow-x-auto">
          <DayTabs days={days} value={day} onChange={onDay} />
        </div>
        {loading ? (
          <div className="flex flex-col gap-8" aria-busy>
            <div className="h-28 animate-pulse rounded-lg bg-surface" />
            <div className="h-48 animate-pulse rounded-xl bg-surface" />
          </div>
        ) : !openWindow ? (
          <EmptyState title="Closed on this day" />
        ) : booked.length === 0 ? (
          <EmptyState title="No bookings yet for this day" />
        ) : (
          <div className="flex flex-col gap-8">
            <LoadStrip reservations={reservations} seats={restaurant.seats} from={openWindow.from} to={openWindow.to} />
            <OwnerTable reservations={reservations} />
          </div>
        )}
      </Page>
    </>
  );
}
