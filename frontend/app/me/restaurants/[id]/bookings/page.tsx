import type { Metadata } from "next";

import { OwnerBookingsRoute } from "@/app/me/restaurants/[id]/bookings/container";

export const metadata: Metadata = { title: "Bookings · Restaurants" };

export default function Page() {
  return <OwnerBookingsRoute />;
}
