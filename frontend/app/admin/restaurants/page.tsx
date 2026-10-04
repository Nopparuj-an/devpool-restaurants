import type { Metadata } from "next";

import { AdminRestaurantsRoute } from "@/app/admin/restaurants/container";

export const metadata: Metadata = { title: "Restaurants · Admin" };

export default function Page() {
  return <AdminRestaurantsRoute />;
}
