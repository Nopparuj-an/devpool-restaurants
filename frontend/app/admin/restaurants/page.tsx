import type { Metadata } from "next";

import { AdminRestaurantsPage } from "@/components/pages/pages";
import { apiGet } from "@/lib/api-server";
import { ADMIN_PAGE_SIZE } from "@/lib/paging";
import type { AdminRestaurant } from "@/lib/types";

import { apiQuery, listParams, requireAdmin } from "../guard";

export const metadata: Metadata = { title: "Restaurants · Admin" };

export default async function AdminRestaurants({ searchParams }: PageProps<"/admin/restaurants">) {
  const account = await requireAdmin("/admin/restaurants");
  const p = listParams(await searchParams);
  const { restaurants, total } = await apiGet<{ restaurants: AdminRestaurant[]; total: number }>(
    `/admin/restaurants?${apiQuery(p, ADMIN_PAGE_SIZE)}`,
  );
  return <AdminRestaurantsPage account={account} restaurants={restaurants} total={total} {...p} />;
}
