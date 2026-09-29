// Page sizes shared by server routes and client screens. Kept out of
// "use client" modules: server code can't read plain values from those.
export const HOME_PAGE_SIZE = 24;
export const REVIEWS_PAGE_SIZE = 20;
export const BOOKINGS_PAGE_SIZE = 50;
export const ADMIN_PAGE_SIZES = [25, 50, 100, 250, 500] as const;
export const ADMIN_PAGE_SIZE = ADMIN_PAGE_SIZES[0];
// The API deletes at most this many rows per request (R-ADMIN-8).
export const ADMIN_DELETE_CHUNK = 500;
