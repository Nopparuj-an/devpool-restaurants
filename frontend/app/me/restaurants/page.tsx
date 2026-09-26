import type { Metadata } from "next";

import { MyRestaurantsScreen } from "@/components/screens/screens";
import { apiGet, requireAccount } from "@/lib/api-server";
import type { RestaurantSummary } from "@/lib/types";

export const metadata: Metadata = { title: "My restaurants · Restaurants" };

export default async function MyRestaurants() {
  const account = await requireAccount("/me/restaurants");
  const { restaurants } = await apiGet<{ restaurants: RestaurantSummary[] }>("/me/restaurants");
  return <MyRestaurantsScreen account={account} restaurants={restaurants} />;
}
