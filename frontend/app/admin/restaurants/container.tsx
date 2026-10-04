"use client";

import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { get } from "@/lib/api-client";
import { keys, useRequireAccount } from "@/lib/queries";
import type { AdminRestaurant } from "@/lib/types";

import { gateAdmin } from "@/app/admin/_shared/gate-admin";
import { AdminRestaurantsScreen } from "./screen";
import { useAdminList, usePageInRange } from "@/app/admin/_shared/use-admin-list";
import { useBanAction } from "@/lib/use-ban-action";
import { useSelection } from "@/app/admin/_shared/use-selection";

export function AdminRestaurantsRoute() {
  const me = useRequireAccount();
  const { apiParams, list, nav } = useAdminList("/admin/restaurants");
  const restaurants = useQuery({
    queryKey: keys.adminRestaurants(apiParams),
    queryFn: () => get<{ restaurants: AdminRestaurant[]; total: number }>(`/admin/restaurants?${apiParams}`),
    enabled: !!me.data?.is_admin,
    placeholderData: keepPreviousData,
  });
  const ban = useBanAction();
  const bulk = useSelection("restaurants");
  usePageInRange(list.page, list.pageSize, restaurants.data?.total, nav.hrefForPage);
  const wait = gateAdmin(me, restaurants);
  if (wait) return wait;
  return (
    <AdminRestaurantsScreen
      account={me.data!}
      {...list}
      {...nav}
      {...bulk}
      restaurants={restaurants.data!.restaurants}
      total={restaurants.data!.total}
      onBan={ban}
    />
  );
}
