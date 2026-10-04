import type { Metadata } from "next";

import { MyBookingsRoute } from "@/app/me/reservations/container";

export const metadata: Metadata = { title: "My bookings · Restaurants" };

export default function Page() {
  return <MyBookingsRoute />;
}
