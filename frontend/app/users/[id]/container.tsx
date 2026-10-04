"use client";

import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useEffect } from "react";

import { get } from "@/lib/api-client";
import { REVIEWS_PAGE_SIZE } from "@/lib/paging";
import { keys, useAccount } from "@/lib/queries";
import { gate, nextOffset, unique, useIdParam } from "@/lib/route-helpers";
import type { Profile, ProfileReviewPage, RestaurantPage } from "@/lib/types";

import { ProfileScreen } from "./screen";

const PROFILE_RESTAURANTS = 12;

export function ProfileRoute() {
  const id = useIdParam();
  const me = useAccount();
  const profile = useQuery({
    queryKey: keys.profile(id),
    queryFn: () => get<Profile>(`/users/${id}`),
  });
  // Their restaurants come from the normal list, filtered by owner.
  const restaurants = useInfiniteQuery({
    queryKey: keys.profileRestaurants(id),
    queryFn: ({ pageParam }) =>
      get<RestaurantPage>(`/restaurants?owner_id=${id}&sort=newest&limit=${PROFILE_RESTAURANTS}&offset=${pageParam}`),
    initialPageParam: 0,
    getNextPageParam: nextOffset<RestaurantPage>((p) => p.restaurants),
  });
  const reviews = useInfiniteQuery({
    queryKey: keys.profileReviews(id),
    queryFn: ({ pageParam }) => get<ProfileReviewPage>(`/users/${id}/reviews?limit=${REVIEWS_PAGE_SIZE}&offset=${pageParam}`),
    initialPageParam: 0,
    getNextPageParam: nextOffset<ProfileReviewPage>((p) => p.reviews),
  });

  const name = profile.data?.display_name;
  useEffect(() => {
    if (name) document.title = `${name} · Restaurants`;
  }, [name]);

  const wait = gate(me, profile, restaurants, reviews);
  if (wait) return wait;
  const rPages = restaurants.data!.pages;
  const vPages = reviews.data!.pages;
  return (
    <ProfileScreen
      account={me.data ?? null}
      profile={profile.data!}
      restaurants={unique(rPages.flatMap((p) => p.restaurants))}
      restaurantsTotal={rPages[rPages.length - 1].total}
      onMoreRestaurants={async () => {
        await restaurants.fetchNextPage();
      }}
      reviews={unique(vPages.flatMap((p) => p.reviews))}
      reviewsTotal={vPages[vPages.length - 1].total}
      onMoreReviews={async () => {
        await reviews.fetchNextPage();
      }}
    />
  );
}
