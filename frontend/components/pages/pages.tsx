"use client";

// Client halves of the real routes: they own the router and API calls and
// hand data and callbacks to the same screens the /design gallery renders.
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";

import type { AuthInput } from "@/components/app/auth-form";
import type { BookingInput } from "@/components/app/booking-panel";
import type { PhotoItem, RestaurantInput } from "@/components/app/restaurant-form";
import { BanDialog, type BanTarget } from "@/components/app/admin";
import {
  AccountScreen,
  AdminRestaurantsScreen,
  AdminUserScreen,
  AdminUsersScreen,
  AuthScreen,
  HomeScreen,
  MyBookingsScreen,
  OwnerBookingsScreen,
  RestaurantEditorScreen,
  RestaurantScreen,
} from "@/components/screens/screens";
import { api } from "@/lib/api-client";
import { dayKey, dayRange, upcomingDays } from "@/lib/days";
import { BOOKINGS_PAGE_SIZE, REVIEWS_PAGE_SIZE } from "@/lib/paging";
import type {
  Account,
  AdminRestaurant,
  AdminStatus,
  AdminUser,
  AdminUserDetail,
  Availability,
  Reservation,
  RestaurantDetail,
  RestaurantImage,
  ReservationPage,
  RestaurantSummary,
  Review,
  ReviewPage,
  SortKey,
} from "@/lib/types";
import { useTimeZone } from "@/lib/use-time-zone";

const EMPTY: Availability = { seats: 0, limited_threshold: 2, limited: false, slots: [] };

export function HomePage(props: {
  account: Account | null;
  restaurants: RestaurantSummary[];
  total: number;
  sort: SortKey;
  query: string;
  page: number;
}) {
  const router = useRouter();
  // Changing search or sort starts again at page 1.
  const href = useCallback(
    (p: { sort?: SortKey; query?: string; page?: number }) => {
      const q = new URLSearchParams();
      const sort = p.sort ?? props.sort;
      const query = p.query ?? props.query;
      if (sort !== "top_rated") q.set("sort", sort);
      if (query) q.set("q", query);
      if (p.page && p.page > 1) q.set("page", String(p.page));
      return q.size ? `/?${q}` : "/";
    },
    [props.sort, props.query],
  );
  const onQuery = useCallback((query: string) => router.replace(href({ query })), [router, href]);
  return (
    <HomeScreen
      {...props}
      onSort={(sort) => router.push(href({ sort }))}
      onQuery={onQuery}
      hrefForPage={(page) => href({ page })}
    />
  );
}

export function RestaurantPage({
  account,
  restaurant,
  reviews: firstPage,
  reviewsTotal,
  myReview,
  editing,
}: {
  account: Account | null;
  restaurant: RestaurantDetail;
  reviews: Review[];
  reviewsTotal: number;
  myReview?: Review;
  editing?: Reservation;
}) {
  const router = useRouter();
  const tz = useTimeZone();
  const [now] = useState(() => new Date());
  const days = useMemo(() => upcomingDays(now, tz, 14), [now, tz]);
  const id = restaurant.id;
  // First page comes from the server; "Show more" appends. A refresh (after
  // writing a review) replaces it with the new first page.
  const [reviews, setReviews] = useState(firstPage);
  const [total, setTotal] = useState(reviewsTotal);
  const [shownFirstPage, setShownFirstPage] = useState(firstPage);
  if (firstPage !== shownFirstPage) {
    setShownFirstPage(firstPage);
    setReviews(firstPage);
    setTotal(reviewsTotal);
  }
  async function onMoreReviews() {
    const res = await api<ReviewPage>("GET", `/restaurants/${id}/reviews?limit=${REVIEWS_PAGE_SIZE}&offset=${reviews.length}`);
    if (res.data) {
      setReviews((rs) => [...rs, ...res.data!.reviews.filter((r) => !rs.some((x) => x.id === r.id))]);
      setTotal(res.data.total);
    }
  }

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
      reviewsTotal={total}
      onMoreReviews={onMoreReviews}
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

export function MyBookingsPage({
  account,
  reservations: firstPage,
  total,
}: {
  account: Account;
  reservations: Reservation[];
  total: number;
}) {
  const router = useRouter();
  const [reservations, setReservations] = useState(firstPage);
  const [shownFirstPage, setShownFirstPage] = useState(firstPage);
  if (firstPage !== shownFirstPage) {
    setShownFirstPage(firstPage);
    setReservations(firstPage);
  }
  return (
    <MyBookingsScreen
      account={account}
      reservations={reservations}
      total={total}
      onMore={async () => {
        const res = await api<ReservationPage>("GET", `/me/reservations?limit=${BOOKINGS_PAGE_SIZE}&offset=${reservations.length}`);
        if (res.data) setReservations((rs) => [...rs, ...res.data!.reservations.filter((r) => !rs.some((x) => x.id === r.id))]);
      }}
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

export function AccountPage({ account }: { account: Account }) {
  const router = useRouter();
  return (
    <AccountScreen
      account={account}
      onSaveName={async (display_name) => {
        const res = await api("PUT", "/me", { display_name });
        if (!res.error) router.refresh(); // header shows the new name
        return { error: res.error };
      }}
      onSetPassword={async (current_password, new_password) => {
        const res = await api("PUT", "/me/password", { current_password, new_password });
        if (res.status === 401) return { error: "Your current password is wrong." };
        if (!res.error) router.refresh(); // has_password may have changed
        return { error: res.error };
      }}
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

// Calls the admin ban / unban endpoint for a user or restaurant, then refreshes.
function useBanAction() {
  const router = useRouter();
  return useCallback(
    async (t: BanTarget, reason: string) => {
      const base = t.kind === "user" ? `/admin/users/${t.id}` : `/admin/restaurants/${t.id}`;
      const res = t.banned ? await api("POST", `${base}/unban`) : await api("POST", `${base}/ban`, { reason });
      if (res.error) window.alert(res.error);
      router.refresh();
    },
    [router],
  );
}

// ?q=&status=&page= for an admin list; changing search or status restarts at page 1.
function useAdminNav(base: string, query: string, status: AdminStatus) {
  const router = useRouter();
  const href = useCallback(
    (p: { query?: string; status?: AdminStatus; page?: number }) => {
      const q = new URLSearchParams();
      const nextQuery = p.query ?? query;
      const nextStatus = p.status ?? status;
      if (nextQuery) q.set("q", nextQuery);
      if (nextStatus) q.set("status", nextStatus);
      if (p.page && p.page > 1) q.set("page", String(p.page));
      return q.size ? `${base}?${q}` : base;
    },
    [base, query, status],
  );
  return {
    onQuery: useCallback((q: string) => router.replace(href({ query: q })), [router, href]),
    onStatus: (s: AdminStatus) => router.push(href({ status: s })),
    hrefForPage: (page: number) => href({ page }),
  };
}

type AdminListPage = { account: Account; total: number; query: string; status: AdminStatus; page: number };

export function AdminUsersPage({ users, ...p }: AdminListPage & { users: AdminUser[] }) {
  const nav = useAdminNav("/admin/users", p.query, p.status);
  return <AdminUsersScreen {...p} {...nav} users={users} onBan={useBanAction()} />;
}

export function AdminRestaurantsPage({ restaurants, ...p }: AdminListPage & { restaurants: AdminRestaurant[] }) {
  const nav = useAdminNav("/admin/restaurants", p.query, p.status);
  return <AdminRestaurantsScreen {...p} {...nav} restaurants={restaurants} onBan={useBanAction()} />;
}

export function AdminUserPage({ account, user }: { account: Account; user: AdminUserDetail }) {
  return <AdminUserScreen account={account} user={user} onBan={useBanAction()} />;
}
