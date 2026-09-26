// Names and titles of the /design screens. Kept out of the client registry so
// server components can read it.
export const SCREENS = [
  { name: "home", title: "Restaurants" },
  { name: "restaurant", title: "Restaurant" },
  { name: "restaurant-guest", title: "Restaurant, logged out" },
  { name: "restaurant-owner", title: "Restaurant, owner view" },
  { name: "booking-edit", title: "Change a booking" },
  { name: "my-bookings", title: "My bookings" },
  { name: "my-bookings-empty", title: "My bookings, empty" },
  { name: "my-restaurants", title: "My restaurants" },
  { name: "my-restaurants-empty", title: "My restaurants, empty" },
  { name: "restaurant-new", title: "Add a restaurant" },
  { name: "restaurant-edit", title: "Edit a restaurant" },
  { name: "owner-bookings", title: "Owner bookings" },
  { name: "admin-users", title: "Admin: users" },
  { name: "admin-user", title: "Admin: one user and their restaurants" },
  { name: "admin-restaurants", title: "Admin: restaurants" },
  { name: "restaurant-hidden", title: "Restaurant hidden by an admin, owner view" },
  { name: "account", title: "Account settings" },
  { name: "account-google", title: "Account settings, Google login only" },
  { name: "login", title: "Log in" },
  { name: "signup", title: "Sign up" },
] as const;

export type ScreenName = (typeof SCREENS)[number]["name"];

export const isScreen = (s: string): s is ScreenName => SCREENS.some((x) => x.name === s);
