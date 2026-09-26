import type { Metadata } from "next";

import { MyBookingsPage } from "@/components/pages/pages";
import { apiGet, requireAccount } from "@/lib/api-server";
import { BOOKINGS_PAGE_SIZE } from "@/lib/paging";
import type { ReservationPage } from "@/lib/types";

export const metadata: Metadata = { title: "My bookings · Restaurants" };

export default async function MyBookings() {
  const account = await requireAccount("/me/reservations");
  const { reservations, total } = await apiGet<ReservationPage>(`/me/reservations?limit=${BOOKINGS_PAGE_SIZE}`);
  return <MyBookingsPage account={account} reservations={reservations} total={total} />;
}
