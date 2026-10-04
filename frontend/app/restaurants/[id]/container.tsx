"use client";

import { keepPreviousData, useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";

import { BanDialog, type BanTarget } from "@/components/app/admin";
import type { BookingInput } from "@/components/app/booking-panel";
import { useBanAction } from "@/lib/use-ban-action";
import { api, get, getOrNull } from "@/lib/api-client";
import { dayKey, dayRange, upcomingDays } from "@/lib/days";
import { REVIEWS_PAGE_SIZE } from "@/lib/paging";
import { keys, useAccount, useRefresh } from "@/lib/queries";
import { gate, nextOffset, unique, useIdParam } from "@/lib/route-helpers";
import type { Account, Availability, Reservation, RestaurantDetail, Review, ReviewPage, ReviewSort } from "@/lib/types";
import { useTimeZone } from "@/lib/use-time-zone";

import { RestaurantScreen } from "./screen";

const EMPTY: Availability = { seats: 0, limited_threshold: 2, limited: false, slots: [] };

export function RestaurantRoute() {
  const id = useIdParam();
  // ?edit=<id> changes one of my bookings here (from My bookings).
  const edit = Number(useSearchParams().get("edit")) || 0;
  const me = useAccount();
  const account = me.data;

  const restaurant = useQuery({
    queryKey: keys.restaurant(id),
    queryFn: () => get<RestaurantDetail>(`/restaurants/${id}`),
  });
  // One star rating or all, newest or oldest first (R-REVIEW-8).
  const [reviewFilter, setReviewFilter] = useState<{ rating: number; sort: ReviewSort }>({ rating: 0, sort: "newest" });
  const reviewParams = new URLSearchParams({ limit: String(REVIEWS_PAGE_SIZE), sort: reviewFilter.sort });
  if (reviewFilter.rating) reviewParams.set("rating", String(reviewFilter.rating));
  const reviews = useInfiniteQuery({
    queryKey: keys.reviews(id, String(reviewParams)),
    queryFn: ({ pageParam }) => get<ReviewPage>(`/restaurants/${id}/reviews?${reviewParams}&offset=${pageParam}`),
    initialPageParam: 0,
    getNextPageParam: nextOffset<ReviewPage>((p) => p.reviews),
    // Keep the page (not a spinner) on screen while a new filter loads.
    placeholderData: keepPreviousData,
  });
  const myReview = useQuery({
    queryKey: keys.myReview(id),
    queryFn: () => getOrNull<Review>(`/restaurants/${id}/reviews/me`),
    enabled: !!account,
  });
  const editing = useQuery({
    queryKey: keys.reservation(edit),
    queryFn: () => getOrNull<Reservation>(`/reservations/${edit}`),
    enabled: !!account && edit > 0,
  });

  const name = restaurant.data?.name;
  useEffect(() => {
    if (name) document.title = `${name} · Restaurants`;
  }, [name]);

  const wait = gate(me, restaurant, reviews, myReview, editing);
  if (wait) return wait;
  const r = restaurant.data!;
  const e = editing.data;
  const pages = reviews.data!.pages;
  return (
    <RestaurantView
      account={account ?? null}
      restaurant={r}
      reviews={unique(pages.flatMap((p) => p.reviews))}
      reviewsTotal={pages[pages.length - 1].total}
      onMoreReviews={async () => {
        await reviews.fetchNextPage();
      }}
      reviewCounts={pages[pages.length - 1].rating_counts}
      reviewFilter={reviewFilter}
      onReviewFilter={setReviewFilter}
      myReview={myReview.data ?? undefined}
      editing={e && e.restaurant.id === r.id && e.can_modify ? e : undefined}
    />
  );
}

function RestaurantView({
  account,
  restaurant,
  reviews,
  reviewsTotal,
  onMoreReviews,
  reviewCounts,
  reviewFilter,
  onReviewFilter,
  myReview,
  editing,
}: {
  account: Account | null;
  restaurant: RestaurantDetail;
  reviews: Review[];
  reviewsTotal: number;
  onMoreReviews: () => Promise<void>;
  reviewCounts: Record<string, number>;
  reviewFilter: { rating: number; sort: ReviewSort };
  onReviewFilter: (f: { rating: number; sort: ReviewSort }) => void;
  myReview?: Review;
  editing?: Reservation;
}) {
  const refresh = useRefresh();
  const tz = useTimeZone();
  const [now] = useState(() => new Date());
  const days = useMemo(() => upcomingDays(now, tz, 14), [now, tz]);
  const id = restaurant.id;

  // The booking panel asks for one day at a time and keeps its own state.
  const loadAvailability = useCallback(
    async (key: string) => {
      const { from, to } = dayRange(key, tz);
      const q = new URLSearchParams({ from, to });
      const res = await api<Availability>("GET", `/restaurants/${id}/availability?${q}`);
      return res.data ?? EMPTY;
    },
    [id, tz],
  );

  async function onBook(input: BookingInput) {
    const res = editing
      ? await api("PUT", `/reservations/${editing.id}`, input)
      : await api("POST", `/restaurants/${id}/reservations`, input);
    if (!res.error) refresh();
    return { error: res.error };
  }

  async function onSaveReview(input: { rating: number; body: string }) {
    const res = await api("PUT", `/restaurants/${id}/reviews/me`, input);
    if (!res.error) await refresh();
    return { error: res.error };
  }

  async function onDeleteReview() {
    await api("DELETE", `/restaurants/${id}/reviews/me`);
    await refresh();
  }

  const [banTarget, setBanTarget] = useState<BanTarget | null>(null);
  const ban = useBanAction();

  return (
    <>
      <RestaurantScreen
        // A different timezone after hydration means different day keys.
        key={tz}
        account={account}
        restaurant={restaurant}
        reviews={reviews}
        reviewsTotal={reviewsTotal}
        onMoreReviews={onMoreReviews}
        reviewCounts={reviewCounts}
        reviewRating={reviewFilter.rating}
        reviewSort={reviewFilter.sort}
        onReviewFilter={onReviewFilter}
        myReview={myReview}
        days={days}
        now={now.toISOString()}
        loadAvailability={loadAvailability}
        onBook={onBook}
        onSaveReview={onSaveReview}
        onDeleteReview={onDeleteReview}
        editing={editing}
        initialDay={editing ? dayKey(new Date(editing.starts_at), tz) : undefined}
        onAdminBan={setBanTarget}
      />
      <BanDialog
        target={banTarget}
        onClose={() => setBanTarget(null)}
        onConfirm={async (t, reason) => {
          await ban(t, reason);
          setBanTarget(null);
        }}
      />
    </>
  );
}
