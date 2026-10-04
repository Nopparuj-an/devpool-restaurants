import { useQuery } from "@tanstack/react-query";

import { NotFoundView } from "@/components/app/states";
import { get } from "@/lib/api-client";
import { keys, useRequireAccount } from "@/lib/queries";
import { gateLoggedIn, useIdParam } from "@/lib/route-helpers";
import type { RestaurantDetail } from "@/lib/types";

// Loads a restaurant for its owner or an admin. Anyone else sees a 404 (the
// API checks every change anyway, R-REST-2, R-ADMIN-6).
export function useOwnedRestaurant() {
  const id = useIdParam();
  const me = useRequireAccount();
  const restaurant = useQuery({
    queryKey: keys.restaurant(id),
    queryFn: () => get<RestaurantDetail>(`/restaurants/${id}`),
    enabled: !!me.data,
  });
  const wait = gateLoggedIn(me, restaurant);
  const denied = !wait && !restaurant.data!.can_manage;
  return {
    wait: denied ? <NotFoundView account={me.data} /> : wait,
    account: me.data!,
    restaurant: restaurant.data!,
  };
}
