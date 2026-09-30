# Design system

Live gallery: **http://localhost:3000/design** (with `pnpm dev` running). Each screen can also be opened on its own at `/design/screens/<name>`. See [ADR-0008](decisions/0008-design-as-react-components.md) for why the design lives in code. The gallery has a left sidebar (Foundations, Components, Screens) that follows your scroll; on phones it collapses into a bar at the top. New specimens need an entry in `app/design/screen-list.ts` matching their `id` in `gallery.tsx`.

The app's top bar (`SiteHeader`) is sticky, including the impersonation banner. Anything else sticky under it, like the booking panel, must offset by the header height (`lg:top-20`).

## Look

Minimal and quiet: white background, black text, thin grey borders, no shadows except the selected segment. The accent is only for things you can act on or that are selected.

| Token | Value | Use |
|---|---|---|
| `accent` | `#A80C8A` | PEA purple (the main brand color on pea.co.th). Primary buttons, links, selection, stars. White text on it is about 6.8:1 (WCAG AA) |
| `accent-hover` / `accent-soft` | `#8C0A73` / `#FBF0F9` | Hover, and selected backgrounds |
| `ink` / `muted` / `faint` | `#0A0A0A` / `#666` / `#9A9A9A` | Text, secondary text, placeholders |
| `line` / `surface` | `#E8E8E8` / `#F7F7F7` | Borders, quiet fills |
| `success` / `warning` / `danger` | green / amber / red | Status only, each with a `-soft` background |

Tokens live in `frontend/app/globals.css` (`@theme`), so Tailwind classes such as `bg-accent` and `text-muted` follow any change there. The font is **Anuphan** (Google Fonts), which covers Thai and Latin with one typeface. Light theme only for now.

## Code layout

```
frontend/
  components/ui/       primitives: Button, Field/Input/Select/Textarea, Badge, Rating, Stars,
                       Segmented, Stepper, RatingInput, ConfirmDialog, EmptyState, Notice, Photo
  components/app/      domain pieces: SiteHeader, RestaurantCard, Gallery, HoursList,
                       BookingPanel, ReviewItem/ReviewForm, ReservationCard, LoadStrip,
                       OwnerTable, RestaurantForm, AuthForm
  components/screens/  full screens, built from props only
  app/design/          gallery, frames, and the mock-data registry
  lib/types.ts         API shapes (mirrors docs/api.md)
  lib/mock.ts          sample data anchored to Thu 1 Oct 2026
  lib/format.ts        dates, times, counts
  lib/use-time-zone.ts viewer timezone (R-TIME-2) without hydration mismatches
```

Screens take data and callbacks as props and never fetch. The design registry passes mock data, and the real routes pass API data of the same shapes. That's how the design and the app stay the same.

## Behaviour worth knowing

- **BookingPanel** offers a start time only if every 15-minute slot the booking covers has room for the party, the same rule the API enforces (R-BOOK-5). When changing a booking, it adds that booking's own seats back first (R-EDIT-2). "Limited seats left" comes from the API's per-day `limited` flag (R-SEATS-1). The API still decides. The panel only saves pointless round trips.
- **Endless scroll**: public lists (home, reviews, My bookings, profiles) grow with `LoadMore`, which loads the next page when its "Show more" button comes near the viewport and stays clickable for keyboards. A list with another list below it (profile restaurants) loads on click only, so the one below stays reachable. Admin lists keep numbered pages.
- **Reviews**: the star breakdown doubles as a filter (click a row, click again to clear); Newest / Oldest sorts. Counts come from the API, not from the loaded page.
- **Bulk delete**: admins tick rows (the header box ticks the page). Admins and yourself have no box (R-ADMIN-8). The confirm dialog names what cascades.
- **Times** are shown in the viewer's timezone. The first render uses Asia/Bangkok so server and client HTML match, then switches to the browser zone.
- **Layout**: the restaurant page stacks intro, booking, then details on phones. On desktop the booking panel is a sticky right column.

## Copy rules

UI text follows the humanizer rules: sentence case everywhere, plain words, no em or en dashes (ranges read "12:00 to 13:00"), no filler or sales tone, short and specific errors ("Only 2 seats left in that time range."). API error messages are shown to users as they are, so they follow the same rules.

## Checking a change

- `pnpm e2e` (in `frontend/`, with `make up`, `make api` and `pnpm dev` running) drives the real app in headless Chrome or Edge. It signs up, books, changes and cancels a table, writes a review, creates and edits a restaurant with photo uploads, checks the owner bookings page, and logs out. With more than one page of data (`make seed-bulk`) it also checks endless scroll on the home page and the review star filter and sort; as admin it keeps a selection across search, page size and page changes, then deletes the run's account with it. It fails on any console error, then cleans up after itself.
- Screens were also checked at 390 and 1280 wide for horizontal overflow.

## Pages

| Route | Screen | Data |
|---|---|---|
| `/` | HomeScreen | `?sort=&q=` handled by the API; the list grows by 24 as you scroll |
| `/restaurants/[id]` | RestaurantScreen | `?edit=<reservation>` switches the booking panel to "Change booking" |
| `/me/reservations` | MyBookingsScreen | login required |
| `/me/restaurants`, `/me/restaurants/new` | MyRestaurantsScreen, RestaurantEditorScreen | login required |
| `/me/restaurants/[id]/edit`, `/bookings` | RestaurantEditorScreen, OwnerBookingsScreen | owner or admin (`can_manage`), otherwise 404 |
| `/users/[id]` | ProfileScreen | public; names on reviews, restaurant pages and the owner table link here |
| `/admin/users`, `/admin/users/[id]`, `/admin/restaurants` | AdminUsersScreen, AdminUserScreen, AdminRestaurantsScreen | admins only (404 otherwise); `?q=&status=&page=&size=` (25 to 500 per page). Lists tick rows for bulk delete; the selection survives page, search and page size changes. The user page renames and impersonates |
| `/me/account` | AccountScreen | change name; change or set a password |
| `/login`, `/signup` | AuthScreen | `?next=` (same-site paths only), `?error=` from Google |

Each `app/**/page.tsx` only sets the title and renders a route component from `components/pages/pages.tsx` ([ADR-0013](decisions/0013-client-side-data-loading.md)). The route component reads the URL, loads data in the browser with TanStack Query (`lib/queries.ts` has the query keys and the shared hooks, `lib/api-client.ts` the fetch helpers, through the `/api` rewrite), and renders a screen. While data loads it shows `PageLoading`; a 404 shows `NotFoundView`; other failures show `PageError` with a retry button (`components/app/states.tsx`). After any write, `useRefresh()` invalidates the cache so what's on screen refetches. Pages that need a login send visitors to `/login?next=…` (`useRequireAccount`).

**Gotcha, hit twice:** a server component can't read plain values (constants, objects) exported from a `"use client"` module. It receives a client reference instead of the value, with no error. Shared values go in plain modules like `lib/paging.ts` or `app/design/screen-list.ts`.
