# Design system

Live gallery: **http://localhost:3000/design** (with `pnpm dev` running). Each screen can also be opened on its own at `/design/screens/<name>`. See [ADR-0008](decisions/0008-design-as-react-components.md) for why the design lives in code.

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

Screens take data and callbacks as props and never fetch. The design registry passes mock data, and the real routes will pass API data of the same shapes. That's how the design and the app stay the same.

## Behaviour worth knowing

- **BookingPanel** offers a start time only if every 15-minute slot the booking covers has room for the party, the same rule the API enforces (R-BOOK-5). When changing a booking, it adds that booking's own seats back first (R-EDIT-2). "Limited seats left" comes from the API's per-day `limited` flag (R-SEATS-1). The API still decides. The panel only saves pointless round trips.
- **Times** are shown in the viewer's timezone. The first render uses Asia/Bangkok so server and client HTML match, then switches to the browser zone.
- **Layout**: the restaurant page stacks intro, booking, then details on phones. On desktop the booking panel is a sticky right column.

## Copy rules

UI text follows the humanizer rules: sentence case everywhere, plain words, no em or en dashes (ranges read "12:00 to 13:00"), no filler or sales tone, short and specific errors ("Only 2 seats left in that time range."). API error messages are shown to users as they are, so they follow the same rules.

## Checking a change

Screens are checked at 390 and 1280 wide for horizontal overflow and console errors. A headless Edge driven over the DevTools protocol takes the screenshots. The script isn't in the repo yet. Add it under `frontend/scripts/` if this becomes routine.
