"use client";

// Client halves of the real routes: they own the router and API calls and
// hand data and callbacks to the same screens the /design gallery renders.
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";

import type { AuthInput } from "@/components/app/auth-form";
import type { BookingInput } from "@/components/app/booking-panel";
import type { PhotoItem, RestaurantInput } from "@/components/app/restaurant-form";
import {
  AuthScreen,
  HomeScreen,
  MyBookingsScreen,
  OwnerBookingsScreen,
  RestaurantEditorScreen,
  RestaurantScreen,
} from "@/components/screens/screens";
import { api } from "@/lib/api-client";
import { dayKey, dayRange, upcomingDays } from "@/lib/days";
import type {
  Account,
  Availability,
  Reservation,
  RestaurantDetail,
  RestaurantImage,
  RestaurantSummary,
  Review,
  SortKey,
} from "@/lib/types";
import { useTimeZone } from "@/lib/use-time-zone";

const EMPTY: Availability = { seats: 0, limited_threshold: 2, limited: false, slots: [] };

export function HomePage(props: { account: Account | null; restaurants: RestaurantSummary[]; sort: SortKey }) {
  const router = useRouter();
  return <HomeScreen {...props} onSort={(s) => router.push(s === "top_rated" ? "/" : `/?sort=${s}`)} />;
}

export function RestaurantPage({
  account,
  restaurant,
  reviews,
  myReview,
  editing,
}: {
  account: Account | null;
  restaurant: RestaurantDetail;
  reviews: Review[];
  myReview?: Review;
  editing?: Reservation;
}) {
  const router = useRouter();
  const tz = useTimeZone();
  const [now] = useState(() => new Date());
  const days = useMemo(() => upcomingDays(now, tz, 14), [now, tz]);
  const id = restaurant.id;

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
    if (!res.error) router.refresh();
    return { error: res.error };
  }

  async function onSaveReview(input: { rating: number; body: string }) {
    const res = await api("PUT", `/restaurants/${id}/reviews/me`, input);
    if (!res.error) router.refresh();
    return { error: res.error };
  }

  async function onDeleteReview() {
    await api("DELETE", `/restaurants/${id}/reviews/me`);
    router.refresh();
  }

  return (
    <RestaurantScreen
      // A different timezone after hydration means different day keys.
      key={tz}
      account={account}
      restaurant={restaurant}
      reviews={reviews}
      myReview={myReview}
      days={days}
      now={now.toISOString()}
      loadAvailability={loadAvailability}
      onBook={onBook}
      onSaveReview={onSaveReview}
      onDeleteReview={onDeleteReview}
      editing={editing}
      initialDay={editing ? dayKey(new Date(editing.starts_at), tz) : undefined}
    />
  );
}

export function MyBookingsPage({ account, reservations }: { account: Account; reservations: Reservation[] }) {
  const router = useRouter();
  return (
    <MyBookingsScreen
      account={account}
      reservations={reservations}
      onCancel={async (r) => {
        await api("POST", `/reservations/${r.id}/cancel`);
        router.refresh();
      }}
      onChange={(r) => router.push(`/restaurants/${r.restaurant.id}?edit=${r.id}`)}
    />
  );
}

export function RestaurantEditorPage({ account, restaurant }: { account: Account; restaurant?: RestaurantDetail }) {
  const router = useRouter();

  async function create(input: RestaurantInput, photos: PhotoItem[]) {
    const form = new FormData();
    // The owner's browser timezone anchors the weekly hours (R-TIME-5).
    form.set("data", JSON.stringify({ ...input, timezone: Intl.DateTimeFormat().resolvedOptions().timeZone }));
    // The API makes the first image the cover.
    for (const p of [...photos].sort((a, b) => Number(b.isCover) - Number(a.isCover))) {
      if (p.file) form.append("images", p.file);
    }
    const res = await api<RestaurantDetail>("POST", "/restaurants", form);
    if (res.data) router.push(`/restaurants/${res.data.id}`);
    return { error: res.error };
  }

  // Details first, then photos: add new ones before removing old ones so the
  // restaurant never drops to zero images (R-REST-1), then set the cover.
  async function update(r: RestaurantDetail, input: RestaurantInput, photos: PhotoItem[]) {
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
    router.refresh();
    return {};
  }

  return (
    <RestaurantEditorScreen
      // Remount after refresh so the form picks up the saved photos.
      key={restaurant?.images.map((i) => i.id).join(",")}
      account={account}
      restaurant={restaurant}
      onSave={(input, photos) => (restaurant ? update(restaurant, input, photos) : create(input, photos))}
      onDelete={
        restaurant
          ? async () => {
              await api("DELETE", `/restaurants/${restaurant.id}`);
              router.push("/me/restaurants");
              router.refresh();
            }
          : undefined
      }
    />
  );
}

export function OwnerBookingsPage({ account, restaurant }: { account: Account; restaurant: RestaurantDetail }) {
  const tz = useTimeZone();
  const [now] = useState(() => new Date());
  const days = useMemo(() => upcomingDays(now, tz, 14), [now, tz]);
  const [day, setDay] = useState(days[0].key);
  const [data, setData] = useState<{
    day: string;
    reservations: Reservation[];
    openWindow: { from: string; to: string } | null;
  } | null>(null);

  useEffect(() => {
    let live = true;
    const { from, to } = dayRange(day, tz);
    const q = new URLSearchParams({ from, to });
    Promise.all([
      api<{ reservations: Reservation[] }>("GET", `/restaurants/${restaurant.id}/reservations?${q}`),
      api<Availability>("GET", `/restaurants/${restaurant.id}/availability?${q}`),
    ]).then(([res, avail]) => {
      if (!live) return;
      // The chart spans the day's open slots (overnight shifts included).
      const slots = avail.data?.slots ?? [];
      const openWindow = slots.length
        ? { from: slots[0].start, to: new Date(Date.parse(slots[slots.length - 1].start) + 15 * 60000).toISOString() }
        : null;
      setData({ day, reservations: res.data?.reservations ?? [], openWindow });
    });
    return () => {
      live = false;
    };
  }, [day, tz, restaurant.id]);

  const current = data?.day === day ? data : null;
  return (
    <OwnerBookingsScreen
      account={account}
      restaurant={restaurant}
      days={days}
      day={day}
      onDay={setDay}
      reservations={current?.reservations ?? []}
      openWindow={current ? current.openWindow : { from: now.toISOString(), to: now.toISOString() }}
      loading={!current}
    />
  );
}

export function AuthPage({
  mode,
  googleEnabled,
  error,
  next,
}: {
  mode: "login" | "signup";
  googleEnabled: boolean;
  error?: string;
  next: string;
}) {
  const router = useRouter();
  async function onSubmit(input: AuthInput) {
    const res = await api("POST", mode === "login" ? "/auth/login" : "/auth/signup", input);
    if (res.error) return { error: res.error };
    router.push(next);
    router.refresh();
    return {};
  }
  return <AuthScreen mode={mode} googleEnabled={googleEnabled} error={error} next={next} onSubmit={onSubmit} />;
}
