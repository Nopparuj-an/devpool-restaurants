"use client";

import { AdminFilters, AdminTabs, RestaurantTable } from "@/components/app/admin";
import { Page, PageTitle, SiteHeader } from "@/components/app/site-header";
import { EmptyState } from "@/components/ui/misc";
import type { AdminRestaurant } from "@/lib/types";

import { useBan, useBulkDelete, type AdminListProps } from "@/app/admin/_shared/list-parts";

export function AdminRestaurantsScreen({ restaurants, ...p }: AdminListProps & { restaurants: AdminRestaurant[] }) {
  const ban = useBan(p.onBan);
  const bulk = useBulkDelete(p, "restaurant");
  return (
    <>
      <SiteHeader account={p.account} current="/admin" />
      <Page>
        <PageTitle title="Admin" subtitle={`${p.total.toLocaleString("en")} ${p.total === 1 ? "restaurant" : "restaurants"}`} />
        <AdminTabs current="restaurants" />
        <AdminFilters
          query={p.query}
          status={p.status}
          placeholder="Search by name, cuisine or owner email"
          onQuery={p.onQuery}
          onStatus={p.onStatus}
        />
        {bulk.bar}
        {restaurants.length === 0 ? (
          <EmptyState title="No restaurants match" />
        ) : (
          <RestaurantTable restaurants={restaurants} onBan={ban.open} selection={p.selection} />
        )}
        {bulk.footer}
        {ban.dialog}
      </Page>
    </>
  );
}
