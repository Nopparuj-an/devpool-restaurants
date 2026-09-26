import type { Metadata } from "next";

import { AdminRestaurantsRoute } from "@/components/pages/pages";

export const metadata: Metadata = { title: "Restaurants · Admin" };

export default function Page() {
  return <AdminRestaurantsRoute />;
}
