import type { Metadata } from "next";

import { MyRestaurantsRoute } from "@/components/pages/pages";

export const metadata: Metadata = { title: "My restaurants · Restaurants" };

export default function Page() {
  return <MyRestaurantsRoute />;
}
