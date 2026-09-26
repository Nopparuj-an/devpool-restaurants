"use client";

// Every screen with mock data, for /design. Callbacks only simulate latency.
import { useCallback, useState } from "react";

import {
  AccountScreen,
  AuthScreen,
  HomeScreen,
  MyBookingsScreen,
  MyRestaurantsScreen,
  OwnerBookingsScreen,
  RestaurantEditorScreen,
  RestaurantScreen,
} from "@/components/screens/screens";
import * as mock from "@/lib/mock";
import type { SortKey } from "@/lib/types";

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
      page={1}
      onSort={setSort}
      onQuery={setQuery}
      hrefForPage={() => "#"}
      limitedIds={[1]}
    />
  );
}

function Restaurant({ asOwner, anonymous, editing }: { asOwner?: boolean; anonymous?: boolean; editing?: boolean }) {
  const loadAvailability = useCallback(async (day: string) => {
    await wait(200);
    return mock.availability(Number(day));
  }, []);
  return (
    <RestaurantScreen
      account={anonymous ? null : asOwner ? mock.owner : mock.account}
      restaurant={asOwner ? mock.ownedRestaurant : mock.restaurants[0]}
      reviews={mock.reviews}
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
  account: { render: () => <AccountScreen account={mock.account} onSaveName={ok} onSetPassword={ok} /> },
  "account-google": { render: () => <AccountScreen account={mock.googleAccount} onSaveName={ok} onSetPassword={ok} /> },
  login: { render: () => <AuthScreen mode="login" googleEnabled onSubmit={ok} /> },
  signup: { render: () => <AuthScreen mode="signup" googleEnabled onSubmit={ok} /> },
} satisfies Record<ScreenName, { render: () => React.ReactNode }>;

export function ScreenView({ name }: { name: ScreenName }) {
  return screens[name].render();
}
