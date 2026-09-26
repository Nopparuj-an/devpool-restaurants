import "server-only";

import { notFound } from "next/navigation";

import { apiGetOrNull, requireAccount } from "@/lib/api-server";
import type { RestaurantDetail } from "@/lib/types";

// Loads a restaurant for its owner. Anyone else gets a 404 (the API enforces
// ownership on every change anyway, R-REST-2).
export async function ownedRestaurant(id: string, path: string) {
  const account = await requireAccount(path);
  const restaurant = await apiGetOrNull<RestaurantDetail>(`/restaurants/${Number(id) || 0}`);
  if (!restaurant?.is_owner) notFound();
  return { account, restaurant };
}
