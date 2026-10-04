"use client";

import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";

import { useOwnedRestaurant } from "@/app/me/restaurants/[id]/use-owned-restaurant";
import { get } from "@/lib/api-client";
import { dayRange, upcomingDays } from "@/lib/days";
import { keys } from "@/lib/queries";
import type { Account, Availability, Reservation, RestaurantDetail } from "@/lib/types";
import { useTimeZone } from "@/lib/use-time-zone";

import { OwnerBookingsScreen } from "./screen";

export function OwnerBookingsRoute() {
  const { wait, account, restaurant } = useOwnedRestaurant();
  if (wait) return wait;
  return <OwnerBookings account={account} restaurant={restaurant} />;
}

function OwnerBookings({ account, restaurant }: { account: Account; restaurant: RestaurantDetail }) {
  const tz = useTimeZone();
  const [now] = useState(() => new Date());
  const days = useMemo(() => upcomingDays(now, tz, 14), [now, tz]);
  const [day, setDay] = useState(days[0].key);
  const { from, to } = dayRange(day, tz);

  const data = useQuery({
    queryKey: keys.ownerDay(restaurant.id, from, to),
    queryFn: async () => {
      const q = new URLSearchParams({ from, to });
      const [res, avail] = await Promise.all([
        get<{ reservations: Reservation[] }>(`/restaurants/${restaurant.id}/reservations?${q}`),
        get<Availability>(`/restaurants/${restaurant.id}/availability?${q}`),
      ]);
      // The chart spans the day's open slots (overnight shifts included).
      const slots = avail.slots;
      const openWindow = slots.length
        ? { from: slots[0].start, to: new Date(Date.parse(slots[slots.length - 1].start) + 15 * 60000).toISOString() }
        : null;
      return { reservations: res.reservations, openWindow };
    },
  });

  return (
    <OwnerBookingsScreen
      account={account}
      restaurant={restaurant}
      days={days}
      day={day}
      onDay={setDay}
      reservations={data.data?.reservations ?? []}
      openWindow={data.data ? data.data.openWindow : { from: now.toISOString(), to: now.toISOString() }}
      loading={!data.data}
    />
  );
}
