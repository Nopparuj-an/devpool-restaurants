import type { Metadata } from "next";

import { EditRestaurantRoute } from "@/app/me/restaurants/_editor/container";

export const metadata: Metadata = { title: "Edit restaurant · Restaurants" };

export default function Page() {
  return <EditRestaurantRoute />;
}
