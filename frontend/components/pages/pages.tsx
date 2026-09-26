"use client";

// The real routes. Each one reads the URL, loads its data in the browser with
// TanStack Query (lib/queries.ts), and hands data and callbacks to the same
// screens the /design gallery renders with mock data. app/**/page.tsx files
// only set the title and render one of these.
import {
  keepPreviousData,
  useInfiniteQuery,
  useQuery,
  useQueryClient,
  type UseQueryResult,
} from "@tanstack/react-query";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";

import type { AuthInput } from "@/components/app/auth-form";
import type { BookingInput } from "@/components/app/booking-panel";
import type { PhotoItem, RestaurantInput } from "@/components/app/restaurant-form";
import { BanDialog, type BanTarget } from "@/components/app/admin";
import { NotFoundView, PageError, PageLoading } from "@/components/app/states";
import {
  AccountScreen,
  AdminRestaurantsScreen,
  AdminUserScreen,
  AdminUsersScreen,
  AuthScreen,
  HomeScreen,
  MyBookingsScreen,
  MyRestaurantsScreen,
  OwnerBookingsScreen,
  RestaurantEditorScreen,
  RestaurantScreen,
} from "@/components/screens/screens";
import { api, ApiError, get, getOrNull } from "@/lib/api-client";
import { dayKey, dayRange, upcomingDays } from "@/lib/days";
import { ADMIN_PAGE_SIZE, BOOKINGS_PAGE_SIZE, HOME_PAGE_SIZE, REVIEWS_PAGE_SIZE } from "@/lib/paging";
import { keys, useAccount, useRefresh, useRequireAccount } from "@/lib/queries";
import type {
  Account,
  AdminRestaurant,
  AdminStatus,
  AdminUser,
  AdminUserDetail,
  Availability,
  Reservation,
  ReservationPage,
  RestaurantDetail,
  RestaurantImage,
  RestaurantPage,
  RestaurantSummary,
  Review,
  ReviewPage,
  SortKey,
} from "@/lib/types";
import { useTimeZone } from "@/lib/use-time-zone";

const EMPTY: Availability = { seats: 0, limited_threshold: 2, limited: false, slots: [] };

// ---------------------------------------------------------------------------
// Loading gates

type Loadable = { isLoading: boolean; isError: boolean; error: unknown; refetch: () => unknown };
type Me = UseQueryResult<Account | null>;

// The screen to show instead of the page while anything is loading or failed,
// or null when everything is ready. A 404 shows the not-found page. Disabled
// queries (enabled: false) count as ready, with no data.
function gate(me: Me, ...queries: Loadable[]): ReactNode {
  const all = [me, ...queries];
  const failed = all.find((q) => q.isError);
  if (failed) {
    if (failed.error instanceof ApiError && failed.error.status === 404) return <NotFoundView account={me.data} />;
    return <PageError account={me.data} onRetry={() => all.forEach((q) => q.isError && q.refetch())} />;
  }
  if (all.some((q) => q.isLoading)) return <PageLoading account={me.data} />;
  return null;
}

// Same, for pages behind a login: keeps showing "loading" while
// useRequireAccount redirects a logged-out visitor.
function gateLoggedIn(me: Me, ...queries: Loadable[]): ReactNode {
  return gate(me, ...queries) ?? (me.data ? null : <PageLoading />);
}

// Offset paging for useInfiniteQuery: the next offset is how many rows are loaded.
function nextOffset<P extends { total: number }>(rows: (p: P) => unknown[]) {
  return (last: P, all: P[]) => {
    const loaded = all.reduce((n, p) => n + rows(p).length, 0);
    return loaded < last.total ? loaded : undefined;
  };
}

// Rows can shift between pages when something is added meanwhile.
function unique<T extends { id: number }>(rows: T[]): T[] {
  const seen = new Set<number>();
  return rows.filter((r) => !seen.has(r.id) && seen.add(r.id));
}

function useIdParam() {
  return Number(useParams<{ id: string }>().id) || 0;
}

// ---------------------------------------------------------------------------
// Home

const SORTS: SortKey[] = ["top_rated", "most_reviewed", "newest"];

// ?sort=&q=&page= are all handled by the API, so the list scales to any size.
export function HomeRoute() {
  const router = useRouter();
  const params = useSearchParams();
  const sort = SORTS.includes(params.get("sort") as SortKey) ? (params.get("sort") as SortKey) : "top_rated";
  const query = (params.get("q") ?? "").trim().slice(0, 100);
  const page = Math.max(1, Math.floor(Number(params.get("page"))) || 1);

  const apiParams = new URLSearchParams({ sort, limit: String(HOME_PAGE_SIZE), offset: String((page - 1) * HOME_PAGE_SIZE) });
  if (query) apiParams.set("q", query);
  const me = useAccount();
  const list = useQuery({
    queryKey: keys.restaurants(String(apiParams)),
    queryFn: () => get<RestaurantPage>(`/restaurants?${apiParams}`),
    // Keep the current page on screen while the next one loads.
    placeholderData: keepPreviousData,
  });

  // Changing search or sort starts again at page 1.
  const href = useCallback(
    (p: { sort?: SortKey; query?: string; page?: number }) => {
      const q = new URLSearchParams();
      const nextSort = p.sort ?? sort;
      const nextQuery = p.query ?? query;
      if (nextSort !== "top_rated") q.set("sort", nextSort);
      if (nextQuery) q.set("q", nextQuery);
      if (p.page && p.page > 1) q.set("page", String(p.page));
      return q.size ? `/?${q}` : "/";
    },
    [sort, query],
  );
  const onQuery = useCallback((q: string) => router.replace(href({ query: q })), [router, href]);

  const wait = gate(me, list);
  if (wait) return wait;
  return (
    <HomeScreen
      account={me.data ?? null}
      restaurants={list.data!.restaurants}
      total={list.data!.total}
      sort={sort}
      query={query}
      page={page}
      onSort={(s) => router.push(href({ sort: s }))}
      onQuery={onQuery}
      hrefForPage={(p) => href({ page: p })}
    />
  );
}

// ---------------------------------------------------------------------------
// Restaurant

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
  const reviews = useInfiniteQuery({
    queryKey: keys.reviews(id),
    queryFn: ({ pageParam }) => get<ReviewPage>(`/restaurants/${id}/reviews?limit=${REVIEWS_PAGE_SIZE}&offset=${pageParam}`),
    initialPageParam: 0,
    getNextPageParam: nextOffset<ReviewPage>((p) => p.reviews),
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
  myReview,
  editing,
}: {
  account: Account | null;
  restaurant: RestaurantDetail;
  reviews: Review[];
  reviewsTotal: number;
  onMoreReviews: () => Promise<void>;
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

// ---------------------------------------------------------------------------
// My bookings and my restaurants

export function MyBookingsRoute() {
  const router = useRouter();
  const refresh = useRefresh();
  const me = useRequireAccount();
  const list = useInfiniteQuery({
    queryKey: keys.myReservations,
    queryFn: ({ pageParam }) => get<ReservationPage>(`/me/reservations?limit=${BOOKINGS_PAGE_SIZE}&offset=${pageParam}`),
    initialPageParam: 0,
    getNextPageParam: nextOffset<ReservationPage>((p) => p.reservations),
    enabled: !!me.data,
  });

  const wait = gateLoggedIn(me, list);
  if (wait) return wait;
  const pages = list.data!.pages;
  return (
    <MyBookingsScreen
      account={me.data!}
      reservations={unique(pages.flatMap((p) => p.reservations))}
      total={pages[0].total}
      onMore={async () => {
        await list.fetchNextPage();
      }}
      onCancel={async (r) => {
        await api("POST", `/reservations/${r.id}/cancel`);
        await refresh();
      }}
      onChange={(r) => router.push(`/restaurants/${r.restaurant.id}?edit=${r.id}`)}
    />
  );
}

export function MyRestaurantsRoute() {
  const me = useRequireAccount();
  const list = useQuery({
    queryKey: keys.myRestaurants,
    queryFn: () => get<{ restaurants: RestaurantSummary[] }>("/me/restaurants"),
    enabled: !!me.data,
  });
  const wait = gateLoggedIn(me, list);
  if (wait) return wait;
  return <MyRestaurantsScreen account={me.data!} restaurants={list.data!.restaurants} />;
}

// Loads a restaurant for its owner. Anyone else sees a 404 (the API enforces
// ownership on every change anyway, R-REST-2).
function useOwnedRestaurant() {
  const id = useIdParam();
  const me = useRequireAccount();
  const restaurant = useQuery({
    queryKey: keys.restaurant(id),
    queryFn: () => get<RestaurantDetail>(`/restaurants/${id}`),
    enabled: !!me.data,
  });
  const wait = gateLoggedIn(me, restaurant);
  const notOwner = !wait && !restaurant.data!.is_owner;
  return {
    wait: notOwner ? <NotFoundView account={me.data} /> : wait,
    account: me.data!,
    restaurant: restaurant.data!,
  };
}

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
    await refresh();
    return {};
  }

  return (
    <RestaurantEditorScreen
      // Remount after a save so the form picks up the saved photos.
      key={restaurant?.images.map((i) => i.id).join(",")}
      account={account}
      restaurant={restaurant}
      onSave={(input, photos) => (restaurant ? update(restaurant, input, photos) : create(input, photos))}
      onDelete={
        restaurant
          ? async () => {
              await api("DELETE", `/restaurants/${restaurant.id}`);
              router.push("/me/restaurants");
              // Nothing under ["restaurant", id] exists any more.
              client.removeQueries({ queryKey: keys.restaurant(restaurant.id) });
              refresh();
            }
          : undefined
      }
    />
  );
}

export function OwnerBookingsRoute() {
  const { wait, account, restaurant } = useOwnedRestaurant();
  if (wait) return wait;
  return <OwnerBookings account={account} restaurant={restaurant} />;
}

function OwnerBookings({ account, restaurant }: { account: Account; restaurant: RestaurantDetail }) {
  const tz = useTimeZone();
  const [now] = useState(() => new Date());
  const days = useMemo(() => upcomingDays(now, tz, 14), [now, tz]);
  const [day, setDay] = useState(days[0].key);
  const { from, to } = dayRange(day, tz);

  const data = useQuery({
    queryKey: keys.ownerDay(restaurant.id, from, to),
    queryFn: async () => {
      const q = new URLSearchParams({ from, to });
      const [res, avail] = await Promise.all([
        get<{ reservations: Reservation[] }>(`/restaurants/${restaurant.id}/reservations?${q}`),
        get<Availability>(`/restaurants/${restaurant.id}/availability?${q}`),
      ]);
      // The chart spans the day's open slots (overnight shifts included).
      const slots = avail.slots;
      const openWindow = slots.length
        ? { from: slots[0].start, to: new Date(Date.parse(slots[slots.length - 1].start) + 15 * 60000).toISOString() }
        : null;
      return { reservations: res.reservations, openWindow };
    },
  });

  return (
    <OwnerBookingsScreen
      account={account}
      restaurant={restaurant}
      days={days}
      day={day}
      onDay={setDay}
      reservations={data.data?.reservations ?? []}
      openWindow={data.data ? data.data.openWindow : { from: now.toISOString(), to: now.toISOString() }}
      loading={!data.data}
    />
  );
}

// ---------------------------------------------------------------------------
// Account and auth

export function AccountRoute() {
  const refresh = useRefresh();
  const me = useRequireAccount();
  const wait = gateLoggedIn(me);
  if (wait) return wait;
  return (
    <AccountScreen
      account={me.data!}
      onSaveName={async (display_name) => {
        const res = await api("PUT", "/me", { display_name });
        if (!res.error) await refresh(); // header shows the new name
        return { error: res.error };
      }}
      onSetPassword={async (current_password, new_password) => {
        const res = await api("PUT", "/me/password", { current_password, new_password });
        if (res.status === 401) return { error: "Your current password is wrong." };
        if (!res.error) await refresh(); // has_password may have changed
        return { error: res.error };
      }}
    />
  );
}

// /login and /signup. Only same-site paths are allowed for ?next=.
export function AuthRoute({ mode }: { mode: "login" | "signup" }) {
  const router = useRouter();
  const refresh = useRefresh();
  const params = useSearchParams();
  const rawNext = params.get("next");
  const next = rawNext && rawNext.startsWith("/") && !rawNext.startsWith("//") ? rawNext : "/";
  const me = useAccount();
  const providers = useQuery({
    queryKey: keys.providers,
    queryFn: () => get<{ google: boolean }>("/auth/providers"),
    staleTime: Infinity,
  });

  // Already logged in, or just logged in: go on to ?next=.
  const loggedIn = !!me.data;
  useEffect(() => {
    if (loggedIn) router.replace(next);
  }, [loggedIn, next, router]);

  async function onSubmit(input: AuthInput) {
    const res = await api("POST", mode === "login" ? "/auth/login" : "/auth/signup", input);
    if (res.error) return { error: res.error };
    // Everything cached so far was loaded logged out. Refetching /me also
    // triggers the redirect above.
    await refresh();
    return {};
  }

  const wait = gate(me, providers);
  if (wait) return wait;
  if (loggedIn) return <PageLoading account={me.data} />;
  return (
    <AuthScreen
      mode={mode}
      googleEnabled={providers.data!.google}
      error={params.get("error") ?? undefined}
      next={next}
      onSubmit={onSubmit}
    />
  );
}

// ---------------------------------------------------------------------------
// Admin

// Calls the admin ban / unban endpoint for a user or restaurant, then refreshes.
function useBanAction() {
  const refresh = useRefresh();
  return useCallback(
    async (t: BanTarget, reason: string) => {
      const base = t.kind === "user" ? `/admin/users/${t.id}` : `/admin/restaurants/${t.id}`;
      const res = t.banned ? await api("POST", `${base}/unban`) : await api("POST", `${base}/ban`, { reason });
      if (res.error) window.alert(res.error);
      await refresh();
    },
    [refresh],
  );
}

// ?q=&status=&page= for an admin list; changing search or status restarts at page 1.
function useAdminList(base: string) {
  const router = useRouter();
  const params = useSearchParams();
  const query = (params.get("q") ?? "").trim().slice(0, 100);
  const rawStatus = params.get("status");
  const status: AdminStatus = rawStatus === "active" || rawStatus === "banned" ? rawStatus : "";
  const page = Math.max(1, Math.floor(Number(params.get("page"))) || 1);

  const apiParams = new URLSearchParams({ limit: String(ADMIN_PAGE_SIZE), offset: String((page - 1) * ADMIN_PAGE_SIZE) });
  if (query) apiParams.set("q", query);
  if (status) apiParams.set("status", status);

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
    apiParams: String(apiParams),
    list: { query, status, page },
    nav: {
      onQuery: useCallback((q: string) => router.replace(href({ query: q })), [router, href]),
      onStatus: (s: AdminStatus) => router.push(href({ status: s })),
      hrefForPage: (p: number) => href({ page: p }),
    },
  };
}

// Admin pages 404 for everyone else (the API also refuses, R-ADMIN-1).
function gateAdmin(me: Me, ...queries: Loadable[]): ReactNode {
  const wait = gateLoggedIn(me, ...queries);
  if (wait) return wait;
  return me.data!.is_admin ? null : <NotFoundView account={me.data} />;
}

export function AdminUsersRoute() {
  const me = useRequireAccount();
  const { apiParams, list, nav } = useAdminList("/admin/users");
  const users = useQuery({
    queryKey: keys.adminUsers(apiParams),
    queryFn: () => get<{ users: AdminUser[]; total: number }>(`/admin/users?${apiParams}`),
    enabled: !!me.data?.is_admin,
    placeholderData: keepPreviousData,
  });
  const ban = useBanAction();
  const wait = gateAdmin(me, users);
  if (wait) return wait;
  return (
    <AdminUsersScreen account={me.data!} {...list} {...nav} users={users.data!.users} total={users.data!.total} onBan={ban} />
  );
}

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
  const wait = gateAdmin(me, restaurants);
  if (wait) return wait;
  return (
    <AdminRestaurantsScreen
      account={me.data!}
      {...list}
      {...nav}
      restaurants={restaurants.data!.restaurants}
      total={restaurants.data!.total}
      onBan={ban}
    />
  );
}

export function AdminUserRoute() {
  const id = useIdParam();
  const me = useRequireAccount();
  const user = useQuery({
    queryKey: keys.adminUser(id),
    queryFn: () => get<AdminUserDetail>(`/admin/users/${id}`),
    enabled: !!me.data?.is_admin,
  });
  const ban = useBanAction();
  const wait = gateAdmin(me, user);
  if (wait) return wait;
  return <AdminUserScreen account={me.data!} user={user.data!} onBan={ban} />;
}
