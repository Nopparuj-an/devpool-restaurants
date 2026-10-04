"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { useState } from "react";

import type { PhotoItem, RestaurantInput } from "@/components/app/restaurant-form";
import { useOwnedRestaurant } from "@/app/me/restaurants/[id]/use-owned-restaurant";
import { api } from "@/lib/api-client";
import { keys, useRefresh, useRequireAccount } from "@/lib/queries";
import { gateLoggedIn } from "@/lib/route-helpers";
import type { Account, RestaurantDetail, RestaurantImage } from "@/lib/types";

import { RestaurantEditorScreen } from "./screen";

export function NewRestaurantRoute() {
  const me = useRequireAccount();
  const wait = gateLoggedIn(me);
  if (wait) return wait;
  return <RestaurantEditor account={me.data!} />;
}

export function EditRestaurantRoute() {
  const { wait, account, restaurant } = useOwnedRestaurant();
  if (wait) return wait;
  return <RestaurantEditor account={account} restaurant={restaurant} />;
}

function RestaurantEditor({ account, restaurant }: { account: Account; restaurant?: RestaurantDetail }) {
  const router = useRouter();
  const client = useQueryClient();
  const refresh = useRefresh();
  // Lives outside the form so "Saved." survives the remount below.
  const [saved, setSaved] = useState(false);

  async function create(input: RestaurantInput, photos: PhotoItem[]) {
    const form = new FormData();
    // The owner's browser timezone anchors the weekly hours (R-TIME-5).
    form.set("data", JSON.stringify({ ...input, timezone: Intl.DateTimeFormat().resolvedOptions().timeZone }));
    // The API makes the first image the cover.
    for (const p of [...photos].sort((a, b) => Number(b.isCover) - Number(a.isCover))) {
      if (p.file) form.append("images", p.file);
    }
    const res = await api<RestaurantDetail>("POST", "/restaurants", form);
    if (res.data) {
      refresh();
      router.push(`/restaurants/${res.data.id}`);
    }
    return { error: res.error };
  }

  // Details first, then photos: add new ones before removing old ones so the
  // restaurant never drops to zero images (R-REST-1), then set the cover.
  async function update(r: RestaurantDetail, input: RestaurantInput, photos: PhotoItem[]) {
    setSaved(false);
    const details = await api("PUT", `/restaurants/${r.id}`, input);
    if (details.error) return { error: details.error };

    const newPhotos = photos.filter((p) => p.file);
    let coverId = photos.find((p) => p.isCover)?.id;
    if (newPhotos.length) {
      const form = new FormData();
      newPhotos.forEach((p) => form.append("images", p.file!));
      const before = new Set(r.images.map((i) => i.id));
      const res = await api<{ images: RestaurantImage[] }>("POST", `/restaurants/${r.id}/images`, form);
      if (res.error) return { error: res.error };
      // New rows come back in upload order, after the existing ones.
      const added = res.data!.images.filter((i) => !before.has(i.id)).sort((a, b) => a.id - b.id);
      const coverIndex = newPhotos.findIndex((p) => p.isCover);
      if (coverIndex >= 0) coverId = added[coverIndex]?.id;
    }
    const kept = new Set(photos.map((p) => p.id).filter(Boolean));
    for (const img of r.images.filter((i) => !kept.has(i.id))) {
      const res = await api("DELETE", `/restaurants/${r.id}/images/${img.id}`);
      if (res.error) return { error: res.error };
    }
    if (coverId && coverId !== r.images.find((i) => i.is_cover)?.id) {
      const res = await api("PUT", `/restaurants/${r.id}/images/${coverId}/cover`);
      if (res.error) return { error: res.error };
    }
    setSaved(true);
    await refresh();
    return {};
  }

  return (
    <RestaurantEditorScreen
      // Remount after a save so the form picks up the saved photos.
      key={restaurant?.images.map((i) => i.id).join(",")}
      account={account}
      restaurant={restaurant}
      saved={saved}
      onSave={(input, photos) => (restaurant ? update(restaurant, input, photos) : create(input, photos))}
      onDelete={
        restaurant
          ? async () => {
              await api("DELETE", `/restaurants/${restaurant.id}`);
              router.push(restaurant.is_owner ? "/me/restaurants" : "/admin/restaurants");
              // Nothing under ["restaurant", id] exists any more.
              client.removeQueries({ queryKey: keys.restaurant(restaurant.id) });
              refresh();
            }
          : undefined
      }
    />
  );
}
