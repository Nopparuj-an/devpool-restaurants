// Mock data for the /design gallery. Shapes match the API (lib/types.ts), so
// the same components render real data later. Everything is anchored to a
// fixed date so previews look the same every time.
import type {
  Account,
  AdminRestaurant,
  AdminUser,
  AdminUserDetail,
  Availability,
  Hours,
  Profile,
  ProfileReview,
  Reservation,
  RestaurantDetail,
  RestaurantSummary,
  Review,
} from "./types";

// Thursday 1 Oct 2026, 09:00 in Bangkok.
export const MOCK_NOW = "2026-10-01T02:00:00Z";

const photo = (n: number) => `/design/photo-${n}.svg`;

// Bangkok wall-clock time on a date offset from MOCK_NOW, as UTC ISO.
export function bkk(dayOffset: number, hhmm: string): string {
  const [h, m] = hhmm.split(":").map(Number);
  const d = new Date(Date.UTC(2026, 9, 1 + dayOffset, h - 7, m));
  return d.toISOString();
}

const daily = (open: string, close: string, closed: number[] = []): Hours[] =>
  [0, 1, 2, 3, 4, 5, 6].filter((d) => !closed.includes(d)).map((weekday) => ({ weekday, open, close }));

export const account: Account = {
  id: 3,
  email: "alice@example.com",
  display_name: "Alice",
  email_verified: false,
  has_password: true,
  is_admin: false,
};

// Signed up with Google and never set a password.
export const googleAccount: Account = {
  id: 9,
  email: "nok@example.com",
  display_name: "Nok",
  email_verified: true,
  has_password: false,
  is_admin: false,
};

export const owner: Account = {
  id: 1,
  email: "somchai@example.com",
  display_name: "Somchai",
  email_verified: true,
  has_password: true,
  is_admin: false,
};

export const restaurants: RestaurantDetail[] = [
  {
    id: 1,
    name: "ครัวริมคลอง",
    cuisine: "Thai",
    location: "Khlong Bang Luang, Bangkok",
    seats: 10,
    rating: 4.7,
    review_count: 126,
    cover_url: photo(1),
    owner: { id: 1, display_name: "Somchai" },
    description:
      "Thai home cooking by the canal. The room is small, so book ahead for lunch. The green curry changes with the season.",
    cancel_cutoff_minutes: 30,
    max_reservation_minutes: 240,
    timezone: "Asia/Bangkok",
    hours: daily("11:00", "22:00", [1]),
    can_manage: false,
    images: [
      { id: 1, url: photo(1), is_cover: true },
      { id: 2, url: photo(3), is_cover: false },
      { id: 3, url: photo(6), is_cover: false },
    ],
    is_owner: false,
  },
  {
    id: 2,
    name: "ส้มตำหน้าตลาด",
    cuisine: "Isan",
    location: "Talad Noi, Bangkok",
    seats: 24,
    rating: 4.6,
    review_count: 311,
    cover_url: photo(2),
    owner: { id: 1, display_name: "Somchai" },
    description: "Som tam, grilled chicken and sticky rice across from the market.",
    cancel_cutoff_minutes: 30,
    max_reservation_minutes: 240,
    timezone: "Asia/Bangkok",
    hours: daily("10:00", "21:00"),
    can_manage: false,
    images: [{ id: 4, url: photo(2), is_cover: true }],
    is_owner: false,
  },
  {
    id: 3,
    name: "ก๋วยเตี๋ยวเรือหน้าวัด",
    cuisine: "Noodles",
    location: "Victory Monument, Bangkok",
    seats: 16,
    rating: 4.3,
    review_count: 58,
    cover_url: photo(3),
    owner: { id: 1, display_name: "Somchai" },
    description: "Boat noodles in small bowls. Most people order three or four.",
    cancel_cutoff_minutes: 30,
    max_reservation_minutes: 120,
    timezone: "Asia/Bangkok",
    hours: daily("09:00", "16:00"),
    can_manage: false,
    images: [{ id: 5, url: photo(3), is_cover: true }],
    is_owner: false,
  },
  {
    id: 4,
    name: "Midnight Moo Kra Ta",
    cuisine: "BBQ",
    location: "Ratchada, Bangkok",
    seats: 30,
    rating: 4.1,
    review_count: 12,
    cover_url: photo(5),
    owner: { id: 2, display_name: "Malee" },
    description: "Thai barbecue hotpot. Opens at six and goes until two in the morning.",
    cancel_cutoff_minutes: 120,
    max_reservation_minutes: 180,
    timezone: "Asia/Bangkok",
    hours: daily("18:00", "02:00"),
    can_manage: false,
    images: [{ id: 6, url: photo(5), is_cover: true }],
    is_owner: false,
  },
  {
    id: 5,
    name: "Sabai 24h Café",
    cuisine: "Café",
    location: "Ari, Bangkok",
    seats: 12,
    rating: null,
    review_count: 0,
    cover_url: photo(4),
    owner: { id: 2, display_name: "Malee" },
    description: "Coffee, toast and khao tom at any hour.",
    cancel_cutoff_minutes: 30,
    max_reservation_minutes: 120,
    timezone: "Asia/Bangkok",
    hours: daily("00:00", "00:00"),
    can_manage: false,
    images: [{ id: 7, url: photo(4), is_cover: true }],
    is_owner: false,
  },
];

export const summaries: RestaurantSummary[] = restaurants.map(
  ({ id, name, cuisine, location, seats, rating, review_count, cover_url, owner }) => ({
    id,
    name,
    cuisine,
    location,
    seats,
    rating,
    review_count,
    cover_url,
    owner,
  }),
);

// The same restaurant as its owner sees it.
export const ownedRestaurant: RestaurantDetail = { ...restaurants[0], is_owner: true, can_manage: true, upcoming_reservations: 12 };

// Availability for ครัวริมคลอง, days from MOCK_NOW. Tomorrow's lunch is almost full.
export function availability(dayOffset: number): Availability {
  const booked: Record<string, number> = {
    "12:00": 8, "12:15": 8, "12:30": 8, "12:45": 8,
    "13:00": 4, "13:15": 4,
    "18:00": 2, "18:15": 2, "18:30": 2, "18:45": 2, "19:00": 6, "19:15": 6, "19:30": 10, "19:45": 10,
  };
  const slots: Availability["slots"] = [];
  // Day 4 is Monday 5 Oct, when ครัวริมคลอง is closed.
  if (dayOffset === 4) return { seats: 10, limited_threshold: 2, limited: false, slots };
  for (let min = 11 * 60; min < 22 * 60; min += 15) {
    const hhmm = `${String(Math.floor(min / 60)).padStart(2, "0")}:${String(min % 60).padStart(2, "0")}`;
    const left = 10 - (dayOffset === 1 ? (booked[hhmm] ?? 0) : Math.floor((booked[hhmm] ?? 0) / 2));
    slots.push({ start: bkk(dayOffset, hhmm), seats_left: left, limited: left <= 2 });
  }
  return { seats: 10, limited_threshold: 2, limited: slots.some((s) => s.limited), slots };
}

export const reviews: Review[] = [
  {
    id: 1,
    rating: 5,
    body: "The green curry is the best I've had in Bangkok. It's a tiny place, so book early.",
    verified: true,
    author: { id: 3, display_name: "Alice" },
    created_at: bkk(-6, "14:00"),
    updated_at: bkk(-6, "14:00"),
  },
  {
    id: 2,
    rating: 5,
    body: "We sat by the canal. Staff remembered us from last time.",
    verified: true,
    author: { id: 4, display_name: "Bob" },
    created_at: bkk(-13, "20:00"),
    updated_at: bkk(-13, "20:00"),
  },
  {
    id: 3,
    rating: 4,
    body: "Lovely food. It gets slow at peak time.",
    verified: false,
    author: { id: 5, display_name: "Carol" },
    created_at: bkk(-20, "19:00"),
    updated_at: bkk(-18, "09:00"),
  },
];

const ref = (r: RestaurantDetail) => ({ id: r.id, name: r.name, cover_url: r.cover_url });

export const myReservations: Reservation[] = [
  {
    id: 11,
    pax: 4,
    starts_at: bkk(1, "12:00"),
    ends_at: bkk(1, "13:00"),
    status: "active",
    state: "upcoming",
    modifiable_until: bkk(1, "11:30"),
    can_modify: true,
    restaurant: ref(restaurants[0]),
  },
  {
    id: 12,
    pax: 6,
    starts_at: bkk(2, "23:00"),
    ends_at: bkk(3, "01:00"),
    status: "active",
    state: "upcoming",
    modifiable_until: bkk(2, "21:00"),
    can_modify: true,
    restaurant: ref(restaurants[3]),
  },
  {
    id: 13,
    pax: 2,
    starts_at: bkk(0, "09:15"),
    ends_at: bkk(0, "10:15"),
    status: "active",
    state: "upcoming",
    modifiable_until: bkk(0, "08:45"),
    can_modify: false,
    restaurant: ref(restaurants[4]),
  },
  {
    id: 14,
    pax: 2,
    starts_at: bkk(-6, "12:00"),
    ends_at: bkk(-6, "13:00"),
    status: "active",
    state: "completed",
    modifiable_until: bkk(-6, "11:30"),
    can_modify: false,
    restaurant: ref(restaurants[0]),
  },
  {
    id: 15,
    pax: 3,
    starts_at: bkk(-3, "11:00"),
    ends_at: bkk(-3, "12:00"),
    status: "cancelled",
    state: "cancelled",
    modifiable_until: bkk(-3, "10:30"),
    can_modify: false,
    restaurant: ref(restaurants[2]),
  },
];

const customer = (id: number, display_name: string) => ({
  id,
  display_name,
  email: `${display_name.toLowerCase()}@example.com`,
});

// The owner's table for ครัวริมคลอง tomorrow.
export const ownerReservations: Reservation[] = [
  ["Alice", 4, "12:00", "13:00"],
  ["Bob", 4, "12:00", "13:30"],
  ["Dan", 2, "18:00", "19:30"],
  ["Carol", 4, "19:00", "20:30"],
  ["Erin", 4, "19:30", "21:00"],
].map(([name, pax, from, to], i) => ({
  id: 100 + i,
  pax: pax as number,
  starts_at: bkk(1, from as string),
  ends_at: bkk(1, to as string),
  status: "active" as const,
  state: "upcoming" as const,
  modifiable_until: bkk(1, from as string),
  can_modify: false,
  restaurant: ref(restaurants[0]),
  customer: customer(3 + i, name as string),
}));

export const admin: Account = {
  id: 7,
  email: "admin@example.com",
  display_name: "Admin",
  email_verified: false,
  has_password: true,
  is_admin: true,
};

const ago = (days: number) => bkk(-days, "10:00");

export const adminUsers: AdminUser[] = [
  { id: 12, email: "spammer@example.com", display_name: "Best Deals 4 U", is_admin: false, banned_at: ago(1), ban_reason: "Posted ads as reviews", created_at: ago(2), restaurants: 0, reviews: 14, reservations: 0 },
  { id: 9, email: "nok@example.com", display_name: "Nok", is_admin: false, banned_at: null, ban_reason: "", created_at: ago(5), restaurants: 0, reviews: 2, reservations: 3 },
  { id: 7, email: "admin@example.com", display_name: "Admin", is_admin: true, banned_at: null, ban_reason: "", created_at: ago(20), restaurants: 0, reviews: 0, reservations: 0 },
  { id: 2, email: "malee@example.com", display_name: "Malee", is_admin: false, banned_at: null, ban_reason: "", created_at: ago(30), restaurants: 2, reviews: 0, reservations: 0 },
  { id: 1, email: "somchai@example.com", display_name: "Somchai", is_admin: false, banned_at: null, ban_reason: "", created_at: ago(31), restaurants: 3, reviews: 0, reservations: 1 },
];

export const adminRestaurants: AdminRestaurant[] = restaurants.map((r, i) => ({
  id: r.id,
  name: r.name,
  cuisine: r.cuisine,
  location: r.location,
  rating: r.rating,
  review_count: r.review_count,
  banned_at: i === 3 ? ago(1) : null,
  ban_reason: i === 3 ? "Photos are from another restaurant" : "",
  created_at: ago(30 - i),
  owner: { id: r.owner.id, display_name: r.owner.display_name, email: `${r.owner.display_name.toLowerCase()}@example.com`, banned: false },
}));

export const adminUserDetail: AdminUserDetail = {
  ...adminUsers[4],
  owned_restaurants: adminRestaurants.filter((r) => r.owner.id === 1),
};

// Public profile of Alice (a reviewer) and Somchai (an owner).
export const profile: Profile = {
  id: 3,
  display_name: "Alice",
  created_at: ago(90),
  restaurant_count: 0,
  review_count: 3,
};

export const profileReviews: ProfileReview[] = [
  { ...reviews[0], restaurant: ref(restaurants[0]) },
  { id: 11, rating: 4, body: "Crispy pork was great. The rice ran out by 8.", verified: false, created_at: ago(20), updated_at: ago(20), restaurant: ref(restaurants[1]) },
  { id: 12, rating: 3, body: "Nice view, slow service.", verified: true, created_at: ago(40), updated_at: ago(40), restaurant: ref(restaurants[2]) },
];

export const ownerProfile: Profile = {
  id: 1,
  display_name: "Somchai",
  created_at: ago(31),
  restaurant_count: 3,
  review_count: 0,
};

// An admin acting as Alice (R-ADMIN-7).
export const impersonated: Account = { ...account, impersonator: { id: admin.id, display_name: admin.display_name } };
