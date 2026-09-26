import type { Metadata } from "next";

import { NewRestaurantRoute } from "@/components/pages/pages";

export const metadata: Metadata = { title: "Add a restaurant · Restaurants" };

export default function Page() {
  return <NewRestaurantRoute />;
}
