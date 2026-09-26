// API shapes, mirroring docs/api.md. Timestamps are UTC ISO strings.

export type Owner = { id: number; display_name: string };

export type RestaurantSummary = {
  id: number;
  name: string;
  cuisine: string;
  location: string;
  seats: number;
  rating: number | null;
  review_count: number;
  cover_url: string;
  owner: Owner;
  // Only the owner and admins ever see hidden restaurants (R-ADMIN-3, -4).
  banned?: boolean;
  owner_banned?: boolean;
};

export type Hours = { weekday: number; open: string; close: string };

export type RestaurantImage = { id: number; url: string; is_cover: boolean };

export type RestaurantDetail = RestaurantSummary & {
  description: string;
  cancel_cutoff_minutes: number;
  max_reservation_minutes: number;
  timezone: string;
  hours: Hours[];
  images: RestaurantImage[];
  is_owner: boolean;
  // Owner or admin: may edit, delete and see bookings (R-ADMIN-6).
  can_manage: boolean;
  upcoming_reservations?: number;
  ban_reason?: string;
};

export type Slot = { start: string; seats_left: number; limited: boolean };

export type Availability = {
  seats: number;
  limited_threshold: number;
  limited: boolean;
  slots: Slot[];
};

export type ReservationState = "upcoming" | "in_progress" | "completed" | "cancelled";

export type Reservation = {
  id: number;
  pax: number;
  starts_at: string;
  ends_at: string;
  status: "active" | "cancelled";
  state: ReservationState;
  modifiable_until: string;
  can_modify: boolean;
  restaurant: { id: number; name: string; cover_url: string };
  customer?: { id: number; display_name: string; email: string };
};

export type Review = {
  id: number;
  rating: number;
  body: string;
  verified: boolean;
  author: { id: number; display_name: string; email?: string };
  created_at: string;
  updated_at: string;
};

export type Account = {
  id: number;
  email: string;
  display_name: string;
  email_verified: boolean;
  has_password: boolean;
  is_admin: boolean;
  // Set while an admin is acting as this account (R-ADMIN-7).
  impersonator?: { id: number; display_name: string };
};

// Public profile (R-PROFILE-*): never the email.
export type Profile = {
  id: number;
  display_name: string;
  created_at: string;
  restaurant_count: number;
  review_count: number;
  banned?: boolean; // only admins ever see a banned profile
};

export type ProfileReview = {
  id: number;
  rating: number;
  body: string;
  verified: boolean;
  created_at: string;
  updated_at: string;
  restaurant: { id: number; name: string; cover_url: string };
  hidden?: boolean; // owner and admins only
};

export type ProfileReviewPage = { reviews: ProfileReview[]; total: number };

export type SortKey = "top_rated" | "most_reviewed" | "newest";

// List responses carry the total number of matches for paging.
export type RestaurantPage = { restaurants: RestaurantSummary[]; total: number };
export type ReviewPage = { reviews: Review[]; total: number };
export type ReservationPage = { reservations: Reservation[]; total: number };

// Admin (R-ADMIN-*)
export type AdminUser = {
  id: number;
  email: string;
  display_name: string;
  is_admin: boolean;
  banned_at: string | null;
  ban_reason: string;
  created_at: string;
  restaurants: number;
  reviews: number;
  reservations: number;
};

export type AdminRestaurant = {
  id: number;
  name: string;
  cuisine: string;
  location: string;
  rating: number | null;
  review_count: number;
  banned_at: string | null;
  ban_reason: string;
  created_at: string;
  owner: { id: number; display_name: string; email: string; banned: boolean };
};

export type AdminUserDetail = AdminUser & { owned_restaurants: AdminRestaurant[] };
export type AdminStatus = "" | "active" | "banned";
