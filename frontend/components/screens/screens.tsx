"use client";

// Full screens, built only from props. The /design gallery feeds them mock
// data; the real routes will feed them API data with the same shapes.
import { Pencil, Plus, Search, Table2 } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";

import { PasswordForm, ProfileForm } from "@/components/app/account-forms";
import {
  AdminFilters,
  AdminTabs,
  BanDialog,
  RestaurantTable,
  StatusBadge,
  UserTable,
  type BanTarget,
} from "@/components/app/admin";
import { AuthForm, type AuthInput } from "@/components/app/auth-form";
import { BookingPanel, DayTabs, type BookingInput } from "@/components/app/booking-panel";
import { LoadStrip, OwnerTable } from "@/components/app/owner-table";
import { ReservationCard } from "@/components/app/reservation-card";
import { RestaurantCard } from "@/components/app/restaurant-card";
import { RestaurantForm, type PhotoItem, type RestaurantInput } from "@/components/app/restaurant-form";
import { Gallery, HoursList } from "@/components/app/restaurant-info";
import { ReviewForm, ReviewItem } from "@/components/app/reviews";
import { Page, PageTitle, SiteHeader } from "@/components/app/site-header";
import { Button, ButtonLink } from "@/components/ui/button";
import { ConfirmDialog, Segmented } from "@/components/ui/controls";
import { Input } from "@/components/ui/field";
import { Badge, EmptyState, Notice, Photo, Rating } from "@/components/ui/misc";
import { Pagination, ShowMore } from "@/components/ui/pagination";
import * as fmt from "@/lib/format";
import type {
  Account,
  AdminRestaurant,
  AdminStatus,
  AdminUser,
  AdminUserDetail,
  Availability,
  Reservation,
  RestaurantDetail,
  RestaurantSummary,
  Review,
  SortKey,
} from "@/lib/types";
import { ADMIN_PAGE_SIZE, HOME_PAGE_SIZE } from "@/lib/paging";
import { useTimeZone } from "@/lib/use-time-zone";

type Result = Promise<{ error?: string }>;
type Day = { key: string; label: string };

const SORTS: { value: SortKey; label: string }[] = [
  { value: "top_rated", label: "Top rated" },
  { value: "most_reviewed", label: "Most reviewed" },
  { value: "newest", label: "New" },
];

// Search, sort and paging all happen on the server; the screen reports
// changes and the route turns them into ?q=&sort=&page=.
export function HomeScreen({
  account,
  restaurants,
  total,
  sort,
  query,
  page,
  onSort,
  onQuery,
  hrefForPage,
  limitedIds = [],
}: {
  account: Account | null;
  restaurants: RestaurantSummary[];
  total: number;
  sort: SortKey;
  query: string;
  page: number;
  onSort: (sort: SortKey) => void;
  onQuery: (query: string) => void;
  hrefForPage: (page: number) => string;
  limitedIds?: number[];
}) {
  const [text, setText] = useState(query);
  // Search 300 ms after typing stops, not on every key.
  useEffect(() => {
    if (text.trim() === query) return;
    const t = setTimeout(() => onQuery(text.trim()), 300);
    return () => clearTimeout(t);
  }, [text, query, onQuery]);

  return (
    <>
      <SiteHeader account={account} current="/" />
      <Page>
        <PageTitle title="Find a table" subtitle="Book a table at a local restaurant, or open your own." />
        <div className="mb-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="relative sm:w-72">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-faint" />
            <Input
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="Search by name or cuisine"
              className="pl-9"
              aria-label="Search restaurants"
            />
          </div>
          <Segmented label="Sort by" options={SORTS} value={sort} onChange={onSort} />
        </div>
        {query && (
          <p className="-mt-4 mb-6 text-sm text-muted">
            {total === 1 ? "1 restaurant" : `${total.toLocaleString("en")} restaurants`} match &quot;{query}&quot;
          </p>
        )}
        {restaurants.length === 0 ? (
          query ? (
            <EmptyState title="Nothing matches that search">Try a different name or cuisine.</EmptyState>
          ) : (
            <EmptyState title="No restaurants yet" />
          )
        ) : (
          <div className="grid gap-x-6 gap-y-10 sm:grid-cols-2 lg:grid-cols-3">
            {restaurants.map((r) => (
              <RestaurantCard key={r.id} restaurant={r} limited={limitedIds.includes(r.id)} />
            ))}
          </div>
        )}
        <Pagination page={page} pageCount={Math.ceil(total / HOME_PAGE_SIZE)} hrefFor={hrefForPage} />
      </Page>
    </>
  );
}

export function RestaurantScreen({
  account,
  restaurant: r,
  reviews,
  myReview,
  days,
  now,
  loadAvailability,
  onBook,
  onSaveReview,
  onDeleteReview,
  editing,
  initialDay,
  reviewsTotal,
  onMoreReviews,
  onAdminBan,
}: {
  account: Account | null;
  restaurant: RestaurantDetail;
  // Admins get ban / unban here too (R-ADMIN-4).
  onAdminBan?: (target: BanTarget) => void;
  reviews: Review[];
  // All reviews count; `reviews` may be the first page only.
  reviewsTotal?: number;
  onMoreReviews?: () => Promise<void>;
  myReview?: Review;
  days: Day[];
  now: string;
  loadAvailability: (day: string) => Promise<Availability>;
  onBook: (input: BookingInput) => Result;
  onSaveReview: (input: { rating: number; body: string }) => Result;
  onDeleteReview: () => Promise<void>;
  editing?: Reservation;
  initialDay?: string;
}) {
  const others = reviews.filter((rv) => rv.id !== myReview?.id);
  const [loadingMore, setLoadingMore] = useState(false);
  return (
    <>
      <SiteHeader account={account} current="/" />
      <Page>
        <div className="flex flex-col gap-8">
          {(r.banned || r.owner_banned) && (
            <Notice tone="warning">
              <span className="font-medium">Hidden from customers.</span>{" "}
              {r.banned
                ? `An admin banned this restaurant${r.ban_reason ? `: ${r.ban_reason}` : "."}`
                : "The owner's account is suspended."}
            </Notice>
          )}
          {account?.is_admin && onAdminBan && !r.is_owner && (
            <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-line px-4 py-3 text-sm">
              <span className="text-muted">Admin</span>
              <span className="flex gap-2">
                <ButtonLink size="sm" variant="secondary" href={`/admin/users/${r.owner.id}`}>
                  Owner
                </ButtonLink>
                <Button
                  size="sm"
                  variant={r.banned ? "secondary" : "danger"}
                  onClick={() => onAdminBan({ kind: "restaurant", id: r.id, name: r.name, banned: !!r.banned })}
                >
                  {r.banned ? "Unban restaurant" : "Ban restaurant"}
                </Button>
              </span>
            </div>
          )}
          {r.is_owner && (
            <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-accent-soft px-4 py-3 text-sm">
              <span>
                This is your restaurant. {r.upcoming_reservations ?? 0} upcoming{" "}
                {r.upcoming_reservations === 1 ? "booking" : "bookings"}.
              </span>
              <span className="flex gap-2">
                <ButtonLink size="sm" variant="secondary" href={`/me/restaurants/${r.id}/bookings`}>
                  <Table2 className="size-4" /> Bookings
                </ButtonLink>
                <ButtonLink size="sm" variant="secondary" href={`/me/restaurants/${r.id}/edit`}>
                  <Pencil className="size-4" /> Edit
                </ButtonLink>
              </span>
            </div>
          )}

          <Gallery images={r.images} name={r.name} />

          {/* Phones: intro, booking, details. Desktop: booking is a sticky right column. */}
          <div className="grid grid-cols-[minmax(0,1fr)] gap-10 lg:grid-cols-[minmax(0,1fr)_24rem]">
            <div className="flex flex-col gap-2 lg:col-start-1">
              <h1 className="text-3xl font-semibold tracking-tight">{r.name}</h1>
              <p className="text-muted">
                {r.cuisine} · {r.location}
              </p>
              <Rating value={r.rating} count={r.review_count} size="lg" />
              <p className="mt-3 max-w-prose leading-relaxed">{r.description}</p>
            </div>

            <aside className="min-w-0 lg:sticky lg:top-6 lg:col-start-2 lg:row-span-2 lg:row-start-1 lg:self-start">
              <BookingPanel
                seats={r.seats}
                maxMinutes={r.max_reservation_minutes}
                cutoffMinutes={r.cancel_cutoff_minutes}
                days={days}
                now={now}
                loadAvailability={loadAvailability}
                onSubmit={onBook}
                editing={editing}
                initialDay={initialDay}
              />
            </aside>

            <div className="flex min-w-0 flex-col gap-10 lg:col-start-1">
              <section className="flex flex-col gap-4">
                <h2 className="font-semibold">Opening hours</h2>
                <HoursList hours={r.hours} />
                <p className="text-sm text-muted">{r.seats} seats</p>
              </section>

              <section className="flex flex-col gap-4">
                <h2 className="font-semibold">Reviews</h2>
                <ReviewForm
                  mode={!account ? "anonymous" : r.is_owner ? "owner" : "customer"}
                  existing={myReview}
                  onSave={onSaveReview}
                  onDelete={onDeleteReview}
                />
                {others.length === 0 ? (
                  <p className="text-sm text-muted">No reviews yet.</p>
                ) : (
                  <div>
                    {others.map((rv) => (
                      <ReviewItem key={rv.id} review={rv} />
                    ))}
                    {onMoreReviews && (
                      <ShowMore
                        shown={reviews.length}
                        total={reviewsTotal ?? reviews.length}
                        busy={loadingMore}
                        onMore={async () => {
                          setLoadingMore(true);
                          await onMoreReviews();
                          setLoadingMore(false);
                        }}
                      />
                    )}
                  </div>
                )}
              </section>
            </div>
          </div>
        </div>
      </Page>
    </>
  );
}

export function MyBookingsScreen({
  account,
  reservations,
  total,
  onMore,
  onCancel,
  onChange,
}: {
  account: Account;
  reservations: Reservation[];
  // All bookings; `reservations` may be the first pages only (current ones come first).
  total?: number;
  onMore?: () => Promise<void>;
  onCancel: (r: Reservation) => Promise<void>;
  onChange: (r: Reservation) => void;
}) {
  const [tab, setTab] = useState<"upcoming" | "past">("upcoming");
  const [cancelling, setCancelling] = useState<Reservation | null>(null);
  const [loadingMore, setLoadingMore] = useState(false);
  const tz = useTimeZone();
  const upcoming = reservations.filter((r) => r.state === "upcoming" || r.state === "in_progress");
  const past = reservations.filter((r) => r.state === "completed" || r.state === "cancelled");
  const list = tab === "upcoming" ? upcoming : past;
  return (
    <>
      <SiteHeader account={account} current="/me/reservations" />
      <Page>
        <PageTitle title="My bookings" />
        <div className="mb-6">
          <Segmented
            label="Show"
            value={tab}
            onChange={setTab}
            options={[
              { value: "upcoming", label: `Upcoming (${upcoming.length})` },
              { value: "past", label: "Past" },
            ]}
          />
        </div>
        {list.length === 0 ? (
          <EmptyState
            title={tab === "upcoming" ? "No upcoming bookings" : "No past bookings yet"}
            action={tab === "upcoming" && <ButtonLink href="/">Find a table</ButtonLink>}
          />
        ) : (
          <div className="flex max-w-2xl flex-col gap-3">
            {list.map((r) => (
              <ReservationCard key={r.id} reservation={r} onCancel={() => setCancelling(r)} onChange={() => onChange(r)} />
            ))}
          </div>
        )}
        {tab === "past" && onMore && (
          <ShowMore
            shown={reservations.length}
            total={total ?? reservations.length}
            busy={loadingMore}
            onMore={async () => {
              setLoadingMore(true);
              await onMore();
              setLoadingMore(false);
            }}
          />
        )}
        <ConfirmDialog
          open={cancelling !== null}
          title="Cancel this booking?"
          confirmLabel="Cancel booking"
          danger
          onClose={() => setCancelling(null)}
          onConfirm={async () => {
            if (cancelling) await onCancel(cancelling);
            setCancelling(null);
          }}
        >
          {cancelling && (
            <>
              {cancelling.restaurant.name}, {fmt.day(cancelling.starts_at, tz)} at {fmt.time(cancelling.starts_at, tz)} for{" "}
              {fmt.guests(cancelling.pax)}. The seats go back to the restaurant.
            </>
          )}
        </ConfirmDialog>
      </Page>
    </>
  );
}

export function MyRestaurantsScreen({ account, restaurants }: { account: Account; restaurants: RestaurantSummary[] }) {
  return (
    <>
      <SiteHeader account={account} current="/me/restaurants" />
      <Page>
        <PageTitle
          title="My restaurants"
          action={
            restaurants.length > 0 && (
              <ButtonLink href="/me/restaurants/new">
                <Plus className="size-4" /> Add restaurant
              </ButtonLink>
            )
          }
        />
        {restaurants.length === 0 ? (
          <EmptyState
            title="You don't have a restaurant yet"
            action={<ButtonLink href="/me/restaurants/new">Add restaurant</ButtonLink>}
          >
            Add one and customers can start booking it right away.
          </EmptyState>
        ) : (
          <div className="flex flex-col divide-y divide-line rounded-xl border border-line">
            {restaurants.map((r) => (
              <div key={r.id} className="flex flex-wrap items-center gap-4 p-4">
                <Photo src={r.cover_url} className="size-16 shrink-0 rounded-lg" />
                <div className="flex min-w-0 flex-1 flex-col gap-1">
                  <span className="flex items-center gap-2">
                    <Link href={`/restaurants/${r.id}`} className="truncate font-medium hover:text-accent">
                      {r.name}
                    </Link>
                    {r.banned && <Badge tone="danger">Hidden by an admin</Badge>}
                  </span>
                  <Rating value={r.rating} count={r.review_count} />
                </div>
                <div className="flex gap-2">
                  <ButtonLink size="sm" variant="secondary" href={`/me/restaurants/${r.id}/bookings`}>
                    Bookings
                  </ButtonLink>
                  <ButtonLink size="sm" variant="ghost" href={`/me/restaurants/${r.id}/edit`}>
                    Edit
                  </ButtonLink>
                </div>
              </div>
            ))}
          </div>
        )}
      </Page>
    </>
  );
}

export function RestaurantEditorScreen({
  account,
  restaurant,
  onSave,
  onDelete,
}: {
  account: Account;
  restaurant?: RestaurantDetail;
  onSave: (input: RestaurantInput, photos: PhotoItem[]) => Result;
  onDelete?: () => Promise<void>;
}) {
  return (
    <>
      <SiteHeader account={account} current="/me/restaurants" />
      <Page>
        <PageTitle title={restaurant ? `Edit ${restaurant.name}` : "Add a restaurant"} />
        <RestaurantForm initial={restaurant} images={restaurant?.images} onSave={onSave} onDelete={onDelete} />
      </Page>
    </>
  );
}

export function OwnerBookingsScreen({
  account,
  restaurant,
  days,
  day,
  onDay,
  reservations,
  openWindow,
  loading,
}: {
  account: Account;
  restaurant: RestaurantDetail;
  days: Day[];
  day: string;
  onDay: (day: string) => void;
  reservations: Reservation[];
  // Opening hours of the selected day, for the load chart.
  openWindow: { from: string; to: string } | null;
  loading?: boolean;
}) {
  const booked = reservations.filter((r) => r.status === "active");
  return (
    <>
      <SiteHeader account={account} current="/me/restaurants" />
      <Page>
        <PageTitle
          title="Bookings"
          subtitle={restaurant.name}
          action={
            <ButtonLink variant="secondary" href={`/me/restaurants/${restaurant.id}/edit`}>
              Edit restaurant
            </ButtonLink>
          }
        />
        <div className="mb-6 overflow-x-auto">
          <DayTabs days={days} value={day} onChange={onDay} />
        </div>
        {loading ? (
          <div className="flex flex-col gap-8" aria-busy>
            <div className="h-28 animate-pulse rounded-lg bg-surface" />
            <div className="h-48 animate-pulse rounded-xl bg-surface" />
          </div>
        ) : !openWindow ? (
          <EmptyState title="Closed on this day" />
        ) : booked.length === 0 ? (
          <EmptyState title="No bookings yet for this day" />
        ) : (
          <div className="flex flex-col gap-8">
            <LoadStrip reservations={reservations} seats={restaurant.seats} from={openWindow.from} to={openWindow.to} />
            <OwnerTable reservations={reservations} />
          </div>
        )}
      </Page>
    </>
  );
}

export function AccountScreen({
  account,
  onSaveName,
  onSetPassword,
}: {
  account: Account;
  onSaveName: (name: string) => Result;
  onSetPassword: (current: string, next: string) => Result;
}) {
  return (
    <>
      <SiteHeader account={account} />
      <Page>
        <PageTitle title="Account" />
        <div className="max-w-3xl">
          <ProfileForm account={account} onSave={onSaveName} />
          <PasswordForm hasPassword={account.has_password} onSave={onSetPassword} />
        </div>
      </Page>
    </>
  );
}

export function AuthScreen({
  mode,
  googleEnabled,
  error,
  next,
  onSubmit,
}: {
  mode: "login" | "signup";
  googleEnabled: boolean;
  error?: string;
  next?: string;
  onSubmit: (input: AuthInput) => Result;
}) {
  return (
    <>
      <SiteHeader account={null} />
      <Page narrow>
        <div className="pt-4 sm:pt-10">
          <AuthForm mode={mode} googleEnabled={googleEnabled} error={error} next={next} onSubmit={onSubmit} />
        </div>
      </Page>
    </>
  );
}

type AdminListProps = {
  account: Account;
  total: number;
  query: string;
  status: AdminStatus;
  page: number;
  onQuery: (q: string) => void;
  onStatus: (s: AdminStatus) => void;
  hrefForPage: (page: number) => string;
  onBan: (target: BanTarget, reason: string) => Promise<void>;
};

// One ban dialog per screen; tables only say which row was clicked.
function useBan(onBan: (target: BanTarget, reason: string) => Promise<void>) {
  const [target, setTarget] = useState<BanTarget | null>(null);
  const dialog = (
    <BanDialog
      target={target}
      onClose={() => setTarget(null)}
      onConfirm={async (t, reason) => {
        await onBan(t, reason);
        setTarget(null);
      }}
    />
  );
  return { open: setTarget, dialog };
}

export function AdminUsersScreen({ users, ...p }: AdminListProps & { users: AdminUser[] }) {
  const ban = useBan(p.onBan);
  return (
    <>
      <SiteHeader account={p.account} current="/admin" />
      <Page>
        <PageTitle title="Admin" subtitle={`${p.total.toLocaleString("en")} ${p.total === 1 ? "user" : "users"}`} />
        <AdminTabs current="users" />
        <AdminFilters query={p.query} status={p.status} placeholder="Search by name or email" onQuery={p.onQuery} onStatus={p.onStatus} />
        {users.length === 0 ? <EmptyState title="No users match" /> : <UserTable users={users} onBan={ban.open} />}
        <Pagination page={p.page} pageCount={Math.ceil(p.total / ADMIN_PAGE_SIZE)} hrefFor={p.hrefForPage} />
        {ban.dialog}
      </Page>
    </>
  );
}

export function AdminRestaurantsScreen({ restaurants, ...p }: AdminListProps & { restaurants: AdminRestaurant[] }) {
  const ban = useBan(p.onBan);
  return (
    <>
      <SiteHeader account={p.account} current="/admin" />
      <Page>
        <PageTitle title="Admin" subtitle={`${p.total.toLocaleString("en")} ${p.total === 1 ? "restaurant" : "restaurants"}`} />
        <AdminTabs current="restaurants" />
        <AdminFilters query={p.query} status={p.status} placeholder="Search by name, cuisine or owner email" onQuery={p.onQuery} onStatus={p.onStatus} />
        {restaurants.length === 0 ? <EmptyState title="No restaurants match" /> : <RestaurantTable restaurants={restaurants} onBan={ban.open} />}
        <Pagination page={p.page} pageCount={Math.ceil(p.total / ADMIN_PAGE_SIZE)} hrefFor={p.hrefForPage} />
        {ban.dialog}
      </Page>
    </>
  );
}

export function AdminUserScreen({
  account,
  user,
  onBan,
}: {
  account: Account;
  user: AdminUserDetail;
  onBan: (target: BanTarget, reason: string) => Promise<void>;
}) {
  const ban = useBan(onBan);
  const tz = useTimeZone();
  const stats = [
    ["Restaurants", user.restaurants],
    ["Reviews", user.reviews],
    ["Bookings", user.reservations],
  ] as const;
  return (
    <>
      <SiteHeader account={account} current="/admin" />
      <Page>
        <Link href="/admin/users" className="mb-4 inline-block text-sm text-muted hover:text-accent">
          ← All users
        </Link>
        <PageTitle
          title={user.display_name}
          subtitle={`${user.email} · joined ${fmt.day(user.created_at, tz)}`}
          action={
            !user.is_admin &&
            user.id !== account.id && (
              <Button
                variant={user.banned_at ? "secondary" : "danger"}
                onClick={() => ban.open({ kind: "user", id: user.id, name: user.display_name, banned: !!user.banned_at })}
              >
                {user.banned_at ? "Unban user" : "Ban user"}
              </Button>
            )
          }
        />
        <div className="mb-10 flex flex-wrap items-center gap-6">
          <StatusBadge bannedAt={user.banned_at} admin={user.is_admin} />
          {stats.map(([label, n]) => (
            <span key={label} className="text-sm">
              <span className="font-semibold tabular-nums">{n}</span> <span className="text-muted">{label.toLowerCase()}</span>
            </span>
          ))}
        </div>
        {user.banned_at && (
          <div className="mb-10">
            <Notice tone="danger">
              Banned {fmt.day(user.banned_at, tz)}
              {user.ban_reason ? `: ${user.ban_reason}` : "."} Their restaurants and reviews are hidden.
            </Notice>
          </div>
        )}
        <h2 className="mb-4 font-semibold">Restaurants they own</h2>
        {user.owned_restaurants.length === 0 ? (
          <EmptyState title="No restaurants" />
        ) : (
          <RestaurantTable restaurants={user.owned_restaurants} onBan={ban.open} showOwner={false} />
        )}
        {ban.dialog}
      </Page>
    </>
  );
}
