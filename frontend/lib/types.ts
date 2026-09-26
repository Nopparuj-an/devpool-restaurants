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
  upcoming_reservations?: number;
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
};

export type SortKey = "top_rated" | "most_reviewed" | "newest";
