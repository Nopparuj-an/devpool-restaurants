import type { Metadata } from "next";

import { MyBookingsPage } from "@/components/pages/pages";
import { apiGet, requireAccount } from "@/lib/api-server";
import type { Reservation } from "@/lib/types";

export const metadata: Metadata = { title: "My bookings · Restaurants" };

export default async function MyBookings() {
  const account = await requireAccount("/me/reservations");
  const { reservations } = await apiGet<{ reservations: Reservation[] }>("/me/reservations");
  return <MyBookingsPage account={account} reservations={reservations} />;
}
