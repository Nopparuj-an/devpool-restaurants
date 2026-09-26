import type { Metadata } from "next";

import { RestaurantEditorPage } from "@/components/pages/pages";
import { requireAccount } from "@/lib/api-server";

export const metadata: Metadata = { title: "Add a restaurant · Restaurants" };

export default async function NewRestaurant() {
  const account = await requireAccount("/me/restaurants/new");
  return <RestaurantEditorPage account={account} />;
}
