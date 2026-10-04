"use client";

import { useInfiniteQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";

import { api, get } from "@/lib/api-client";
import { BOOKINGS_PAGE_SIZE } from "@/lib/paging";
import { keys, useRefresh, useRequireAccount } from "@/lib/queries";
import { gateLoggedIn, nextOffset, unique } from "@/lib/route-helpers";
import type { ReservationPage } from "@/lib/types";

import { MyBookingsScreen } from "./screen";

export function MyBookingsRoute() {
  const router = useRouter();
  const refresh = useRefresh();
  const me = useRequireAccount();
  const list = useInfiniteQuery({
    queryKey: keys.myReservations,
    queryFn: ({ pageParam }) => get<ReservationPage>(`/me/reservations?limit=${BOOKINGS_PAGE_SIZE}&offset=${pageParam}`),
    initialPageParam: 0,
    getNextPageParam: nextOffset<ReservationPage>((p) => p.reservations),
    enabled: !!me.data,
  });

  const wait = gateLoggedIn(me, list);
  if (wait) return wait;
  const pages = list.data!.pages;
  return (
    <MyBookingsScreen
      account={me.data!}
      reservations={unique(pages.flatMap((p) => p.reservations))}
      total={pages[0].total}
      onMore={async () => {
        await list.fetchNextPage();
      }}
      onCancel={async (r) => {
        await api("POST", `/reservations/${r.id}/cancel`);
        await refresh();
      }}
      onChange={(r) => router.push(`/restaurants/${r.restaurant.id}?edit=${r.id}`)}
    />
  );
}
