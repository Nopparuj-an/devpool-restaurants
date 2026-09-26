import type { Metadata } from "next";

import { OwnerBookingsRoute } from "@/components/pages/pages";

export const metadata: Metadata = { title: "Bookings · Restaurants" };

export default function Page() {
  return <OwnerBookingsRoute />;
}
