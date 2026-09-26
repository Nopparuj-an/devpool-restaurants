import type { Metadata } from "next";

import { EditRestaurantRoute } from "@/components/pages/pages";

export const metadata: Metadata = { title: "Edit restaurant · Restaurants" };

export default function Page() {
  return <EditRestaurantRoute />;
}
