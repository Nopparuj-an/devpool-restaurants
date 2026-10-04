"use client";

import { RestaurantForm, type PhotoItem, type RestaurantInput } from "@/components/app/restaurant-form";
import { Page, PageTitle, SiteHeader } from "@/components/app/site-header";
import type { Account, ActionResult, RestaurantDetail } from "@/lib/types";

export function RestaurantEditorScreen({
  account,
  restaurant,
  saved,
  onSave,
  onDelete,
}: {
  account: Account;
  restaurant?: RestaurantDetail;
  saved?: boolean;
  onSave: (input: RestaurantInput, photos: PhotoItem[]) => ActionResult;
  onDelete?: () => Promise<void>;
}) {
  return (
    <>
      <SiteHeader account={account} current="/me/restaurants" />
      <Page>
        <PageTitle title={restaurant ? `Edit ${restaurant.name}` : "Add a restaurant"} />
        <RestaurantForm initial={restaurant} images={restaurant?.images} saved={saved} onSave={onSave} onDelete={onDelete} />
      </Page>
    </>
  );
}
