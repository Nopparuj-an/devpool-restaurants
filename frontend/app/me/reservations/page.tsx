import type { Metadata } from "next";

import { MyBookingsRoute } from "@/components/pages/pages";

export const metadata: Metadata = { title: "My bookings · Restaurants" };

export default function Page() {
  return <MyBookingsRoute />;
}
