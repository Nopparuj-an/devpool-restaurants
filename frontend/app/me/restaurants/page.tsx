import type { Metadata } from "next";

import { MyRestaurantsRoute } from "@/app/me/restaurants/container";

export const metadata: Metadata = { title: "My restaurants · Restaurants" };

export default function Page() {
  return <MyRestaurantsRoute />;
}
