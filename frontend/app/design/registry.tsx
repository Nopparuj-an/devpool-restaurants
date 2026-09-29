"use client";

// Every screen with mock data, for /design. Callbacks only simulate latency.
import { useCallback, useState } from "react";

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
  ProfileScreen,
  RestaurantEditorScreen,
  RestaurantScreen,
} from "@/components/screens/screens";
import * as mock from "@/lib/mock";
import type { ReviewSort, SortKey } from "@/lib/types";

import type { ScreenName } from "./screen-list";

const wait = (ms = 400) => new Promise<void>((r) => setTimeout(r, ms));
const done = () => wait();
const ok = async () => {
  await wait();
  return {};
};

export const DAYS = ["Today", "Tomorrow", "Sat 3", "Sun 4", "Mon 5", "Tue 6", "Wed 7"].map((label, i) => ({
  key: String(i),
  label,
}));

function Home() {
  const [sort, setSort] = useState<SortKey>("top_rated");
  const [query, setQuery] = useState("");
  const q = query.toLowerCase();
  const sorted = [...mock.summaries]
    .filter((r) => !q || r.name.toLowerCase().includes(q) || r.cuisine.toLowerCase().includes(q))
    .sort((a, b) =>
      sort === "most_reviewed" ? b.review_count - a.review_count : sort === "newest" ? b.id - a.id : (b.rating ?? 0) - (a.rating ?? 0),
    );
  return (
    <HomeScreen
      account={mock.account}
      restaurants={sorted}
      total={sorted.length}
      sort={sort}
      query={query}
      onSort={setSort}
      onQuery={setQuery}
      limitedIds={[1]}
    />
  );
}

// Per-star counts of the mock reviews, like the API's rating_counts.
const reviewCounts = Object.fromEntries([1, 2, 3, 4, 5].map((n) => [n, mock.reviews.filter((r) => r.rating === n).length]));

function Restaurant({ asOwner, anonymous, editing }: { asOwner?: boolean; anonymous?: boolean; editing?: boolean }) {
  const loadAvailability = useCallback(async (day: string) => {
    await wait(200);
    return mock.availability(Number(day));
  }, []);
  const [filter, setFilter] = useState<{ rating: number; sort: ReviewSort }>({ rating: 0, sort: "newest" });
  const reviews = mock.reviews
    .filter((r) => !filter.rating || r.rating === filter.rating)
    .sort((a, b) => (filter.sort === "oldest" ? 1 : -1) * a.updated_at.localeCompare(b.updated_at));
  const r = asOwner ? mock.ownedRestaurant : mock.restaurants[0];
  return (
    <RestaurantScreen
      account={anonymous ? null : asOwner ? mock.owner : mock.account}
      restaurant={{ ...r, review_count: mock.reviews.length }}
      reviews={reviews}
      reviewsTotal={reviews.length}
      reviewCounts={reviewCounts}
      reviewRating={filter.rating}
      reviewSort={filter.sort}
      onReviewFilter={setFilter}
      myReview={asOwner || anonymous ? undefined : mock.reviews[0]}
      days={DAYS.slice(1)}
      now={mock.MOCK_NOW}
      loadAvailability={loadAvailability}
      onBook={ok}
      onSaveReview={ok}
      onDeleteReview={done}
      editing={editing ? mock.myReservations[0] : undefined}
    />
  );
}

// The owner's view of a restaurant an admin banned.
function HiddenRestaurant() {
  const loadAvailability = useCallback(async (day: string) => mock.availability(Number(day)), []);
  return (
    <RestaurantScreen
      account={mock.owner}
      restaurant={{ ...mock.ownedRestaurant, banned: true, ban_reason: "Photos are from another restaurant" }}
      reviews={mock.reviews}
      days={DAYS.slice(1)}
      now={mock.MOCK_NOW}
      loadAvailability={loadAvailability}
      onBook={ok}
      onSaveReview={ok}
      onDeleteReview={done}
    />
  );
}

function OwnerBookings() {
  const [day, setDay] = useState("1");
  const d = Number(day);
  return (
    <OwnerBookingsScreen
      account={mock.owner}
      restaurant={mock.ownedRestaurant}
      days={DAYS}
      day={day}
      onDay={setDay}
      reservations={d === 1 ? mock.ownerReservations : []}
      openWindow={d === 4 ? null : { from: mock.bkk(d, "11:00"), to: mock.bkk(d, "22:00") }}
    />
  );
}

// Admin lists with a working selection; the first row starts ticked.
function useMockSelection(first: { id: number; name: string }) {
  const [selected, setSelected] = useState<ReadonlyMap<number, string>>(() => new Map([[first.id, first.name]]));
  const [pageSize, setPageSize] = useState(25);
  return {
    pageSize,
    onPageSize: setPageSize,
    selection: {
      selected,
      onSelect: (rows: { id: number; name: string }[], on: boolean) =>
        setSelected((prev) => {
          const next = new Map(prev);
          rows.forEach((r) => (on ? next.set(r.id, r.name) : next.delete(r.id)));
          return next;
        }),
    },
    onClearSelection: () => setSelected(new Map()),
    onDeleteSelected: async () => {
      await wait();
      return { error: "This is the design gallery. Nothing was deleted." };
    },
  };
}

function AdminUsers() {
  const bulk = useMockSelection({ id: mock.adminUsers[0].id, name: mock.adminUsers[0].display_name });
  return (
    <AdminUsersScreen
      account={mock.admin}
      users={mock.adminUsers}
      total={mock.adminUsers.length}
      query=""
      status=""
      page={1}
      onQuery={() => {}}
      onStatus={() => {}}
      hrefForPage={() => "#"}
      onBan={done}
      {...bulk}
    />
  );
}

function AdminRestaurants() {
  const bulk = useMockSelection({ id: mock.adminRestaurants[0].id, name: mock.adminRestaurants[0].name });
  return (
    <AdminRestaurantsScreen
      account={mock.admin}
      restaurants={mock.adminRestaurants}
      total={mock.adminRestaurants.length}
      query=""
      status=""
      page={1}
      onQuery={() => {}}
      onStatus={() => {}}
      hrefForPage={() => "#"}
      onBan={done}
      {...bulk}
    />
  );
}

const screens = {
  home: { render: () => <Home /> },
  restaurant: { render: () => <Restaurant /> },
  "restaurant-guest": { render: () => <Restaurant anonymous /> },
  "restaurant-owner": { render: () => <Restaurant asOwner /> },
  "booking-edit": { render: () => <Restaurant editing /> },
  "my-bookings": {
    render: () => <MyBookingsScreen account={mock.account} reservations={mock.myReservations} onCancel={done} onChange={() => {}} />,
  },
  "my-bookings-empty": {
    render: () => <MyBookingsScreen account={mock.account} reservations={[]} onCancel={done} onChange={() => {}} />,
  },
  "my-restaurants": {
    render: () => <MyRestaurantsScreen account={mock.owner} restaurants={mock.summaries.slice(0, 3)} />,
  },
  "my-restaurants-empty": {
    render: () => <MyRestaurantsScreen account={mock.account} restaurants={[]} />,
  },
  "restaurant-new": {
    render: () => <RestaurantEditorScreen account={mock.owner} onSave={ok} />,
  },
  "restaurant-edit": {
    render: () => <RestaurantEditorScreen account={mock.owner} restaurant={mock.ownedRestaurant} onSave={ok} onDelete={done} />,
  },
  "owner-bookings": { render: () => <OwnerBookings /> },
  "admin-users": { render: () => <AdminUsers /> },
  profile: {
    render: () => (
      <ProfileScreen
        account={mock.account}
        profile={mock.profile}
        restaurants={[]}
        restaurantsTotal={0}
        reviews={mock.profileReviews}
        reviewsTotal={mock.profileReviews.length}
      />
    ),
  },
  "profile-owner": {
    render: () => (
      <ProfileScreen
        account={mock.admin}
        profile={mock.ownerProfile}
        restaurants={mock.summaries.slice(0, 3)}
        restaurantsTotal={3}
        reviews={[]}
        reviewsTotal={0}
      />
    ),
  },
  impersonating: {
    render: () => <MyBookingsScreen account={mock.impersonated} reservations={mock.myReservations} onCancel={done} onChange={() => {}} />,
  },
  "admin-user": {
    render: () => (
      <AdminUserScreen account={mock.admin} user={mock.adminUserDetail} onBan={done} onRename={ok} onImpersonate={ok} />
    ),
  },
  "admin-restaurants": { render: () => <AdminRestaurants /> },
  "restaurant-hidden": {
    render: () => <HiddenRestaurant />,
  },
  account: { render: () => <AccountScreen account={mock.account} onSaveName={ok} onSetPassword={ok} /> },
  "account-google": { render: () => <AccountScreen account={mock.googleAccount} onSaveName={ok} onSetPassword={ok} /> },
  login: { render: () => <AuthScreen mode="login" googleEnabled onSubmit={ok} /> },
  signup: { render: () => <AuthScreen mode="signup" googleEnabled onSubmit={ok} /> },
} satisfies Record<ScreenName, { render: () => React.ReactNode }>;

export function ScreenView({ name }: { name: ScreenName }) {
  return screens[name].render();
}
