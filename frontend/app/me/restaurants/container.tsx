"use client";

import { useQuery } from "@tanstack/react-query";

import { get } from "@/lib/api-client";
import { keys, useRequireAccount } from "@/lib/queries";
import { gateLoggedIn } from "@/lib/route-helpers";
import type { RestaurantSummary } from "@/lib/types";

import { MyRestaurantsScreen } from "./screen";

export function MyRestaurantsRoute() {
  const me = useRequireAccount();
  const list = useQuery({
    queryKey: keys.myRestaurants,
    queryFn: () => get<{ restaurants: RestaurantSummary[] }>("/me/restaurants"),
    enabled: !!me.data,
  });
  const wait = gateLoggedIn(me, list);
  if (wait) return wait;
  return <MyRestaurantsScreen account={me.data!} restaurants={list.data!.restaurants} />;
}
