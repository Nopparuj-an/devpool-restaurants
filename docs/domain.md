# Domain: glossary and business rules

The Go backend enforces every rule on this page. Rule IDs (e.g. `R-BOOK-3`) are stable: never renumber them, only add new ones. Reference them in code comments, tests, and error codes.

## Glossary

| Term | Meaning |
|---|---|
| **Account** | A person who logged in. Can own restaurants and book or review any restaurant. |
| **Owner** | The account that created a restaurant. This is a relationship, not a role flag. |
| **Customer** | Any account making a reservation or review. An owner can be a customer of their own restaurant (bookings only, not reviews). |
| **Seats** | Restaurant capacity counted in people (pax), not tables. 10 seats = 10 people at the same moment. |
| **Pax** | Number of people in a reservation. |
| **Slot** | A 15-minute grid step. All times a user picks are on this grid. |
| **Shift** | One opening window, e.g. Fri 18:00 → Sat 02:00. It belongs to the weekday it opens on. |
| **Open time** | The union of all shifts. Shifts that touch or overlap merge into one continuous open period. |
| **Cancel cutoff** | Minutes before a reservation starts after which the customer can no longer edit or cancel it. |
| **Booking horizon** | How far ahead a reservation may start: 30 days. |
| **Load(t)** | Total pax of active reservations covering instant `t`. |
| **Seats left** | `seats − peak Load` over a given window. |
| **Verified review** | A review whose author has had a completed reservation at that restaurant. |

## Time

- **R-TIME-1:** The backend runs in UTC. Instants (reservation start/end, created_at, …) are stored as UTC `timestamptz`, and the API only exchanges UTC ISO-8601 instants.
- **R-TIME-2:** The frontend converts between UTC and the **viewer's browser timezone** for display and input.
- **R-TIME-3:** Every time interval is half-open, `[start, end)`. A 12:00–13:00 booking and a 13:00–14:00 booking do not overlap.
- **R-TIME-4:** The UI shows Gregorian dates in ISO or Thai-locale format. When parsing input, never mistake Buddhist-era years (2569) for Gregorian ones.
- **R-TIME-5:** Each restaurant stores an IANA `timezone`, **captured automatically from the owner's browser** when the restaurant is created. It is never shown in the UI. It exists only so that weekly wall-clock opening hours ("Mon 18:00") can be turned into UTC instants. See [ADR-0007](decisions/0007-time-storage-and-overnight-hours.md).

## Restaurants

- **R-REST-1:** Required fields: name, description, cuisine type, location, seats (≥ 1), at least one image (one is the **cover**), opening hours.
- **R-REST-2:** Only the owner (or an admin, R-ADMIN-6) can update or delete the restaurant. Anyone else gets `403`.
- **R-REST-3:** `cancel_cutoff_minutes` defaults to 30. It must be **≥ 30** and a multiple of 15.
- **R-REST-4:** Changing seats, hours, cutoff, or max duration **never modifies existing reservations**. New settings apply to new bookings and to edits made afterwards.
- **R-REST-5:** Deleting a restaurant **cascades**: its images, reservations (including future ones), and reviews are deleted. The frontend must show a confirmation that says how many upcoming reservations will be lost.
- **R-REST-6:** `max_reservation_minutes` defaults to 240 (4h). It must be ≥ 15 (one slot) and a multiple of 15.

### Opening hours

- **R-HOURS-1:** Hours are set per weekday, with at most one shift per weekday. A weekday with no shift means closed.
- **R-HOURS-2:** `open_time` and `close_time` are wall-clock times in the restaurant's timezone (R-TIME-5), on the 15-minute grid.
- **R-HOURS-3:** `close_time < open_time` means the shift **ends the next day** (18:00–02:00). `close_time == open_time` means the shift is **open 24h** from `open_time`. A restaurant open every day with 00:00–00:00 is open all the time.
- **R-HOURS-4:** A reservation must be **entirely inside open time**. To check, turn the shifts opening on the start date and the day before into UTC instants, merge the ones that touch or overlap, and check that `[starts_at, ends_at)` fits inside one merged period. A 23:00–01:00 booking at a 24h restaurant is valid even though it crosses two shifts.

## Reservations

The customer picks pax, date, and times in their browser's timezone. The frontend sends `pax`, `starts_at`, and `ends_at` as UTC instants.

### Creating (all must hold, otherwise the request is rejected)

- **R-BOOK-1:** `pax ≥ 1` and `pax ≤ seats`.
- **R-BOOK-2:** `starts_at` and `ends_at` are on the 15-minute grid, `ends_at > starts_at`, and `ends_at − starts_at ≤ max_reservation_minutes`.
- **R-BOOK-3:** `starts_at > now`. No bookings in the past.
- **R-BOOK-4:** The reservation fits entirely in open time (R-HOURS-4).
- **R-BOOK-5 (capacity):** For every instant `t` in `[starts_at, ends_at)`: `Load(t) + pax ≤ seats`.
  - Summing all overlapping reservations is **wrong**. Compute the **peak** load within the window: take the start/end events of the overlapping active reservations, sweep through them in time order, and track the running maximum.
  - Example from the brief: 10 seats, A=7 at 12:00–12:30, B=7 at 12:30–13:00. C=3 for 12:00–13:00 fits, because the peak is 7 + 3 = 10, even though 7 + 7 + 3 = 17.
- **R-BOOK-6 (concurrency):** The capacity check and the insert happen in one DB transaction that first locks the restaurant row. Bookings for the same restaurant are therefore serialized. See [ADR-0003](decisions/0003-booking-concurrency.md).
- **R-BOOK-7:** Owners may book their own restaurant.
- **R-BOOK-8:** `starts_at ≤ now + 30 days` (booking horizon).

### Editing

- **R-EDIT-1:** Only the reservation's customer can edit it, and only while it is active and `now ≤ starts_at − cancel_cutoff`.
- **R-EDIT-2:** The new values must pass R-BOOK-1…8. The reservation being edited is **excluded** from `Load(t)` so its old seats aren't counted twice. With 10 seats, A=7 and B=3: A → 8 fails and A → 5 succeeds.
- **R-EDIT-3 (shrink exception):** An edit that **doesn't increase the reservation's footprint** (new pax ≤ old pax, and the new window lies inside the old one) skips R-BOOK-5. It can only lower the load, so it is allowed even when the restaurant is over capacity because the owner lowered seats (R-REST-4). All other rules still apply.

### Cancelling

- **R-CANCEL-1:** Only the reservation's customer can cancel it, while `now ≤ starts_at − cancel_cutoff`, using the restaurant's **current** cutoff. Example: a 12:00 booking with a 30-minute cutoff can be cancelled until 11:30.
- **R-CANCEL-2:** Cancelling sets `status = cancelled`. The row is kept for history. Cancelled reservations never count toward load.

### Derived states

- **Upcoming:** active and `starts_at > now`. **Completed:** active and `ends_at ≤ now`. These are computed at read time, not stored.

### Limited seats left

- **R-SEATS-1:** Show a **Limited Seats Left** badge when, in any 15-minute slot of the day or window being viewed, seats left is **≤ 2 or ≤ 20% of seats**, whichever is larger. Example: 10 seats → badge at ≤ 2 left, 30 seats → badge at ≤ 6 left.

## Reviews

- **R-REVIEW-1:** Rating is an integer from 1 to 5, and review text is required.
- **R-REVIEW-2:** At most **one review per (account, restaurant)**, enforced by a unique constraint. The author can edit it later. Writing again updates the existing review instead of adding a new one.
- **R-REVIEW-3:** Owners cannot review their own restaurant (`403`).
- **R-REVIEW-4:** No reservation is needed to review. A review is shown as **verified** if the author has at least one **completed** reservation at the restaurant (see Derived states). This is computed at read time, so a review becomes verified once a booking completes.
- **R-REVIEW-5:** Restaurants display `average = rating_sum / review_count`, rounded to one decimal, plus the count. With 0 reviews, show "No reviews yet" instead of an average.
- **R-REVIEW-6:** **Most reviewed** sorts by `review_count` descending. **Highest rated** sorts by the Bayesian score in [ADR-0004](decisions/0004-rating-aggregate-and-ranking.md), not by the raw average. Ties are broken by count, then name.
- **R-REVIEW-7:** The author can delete their review. The restaurant's aggregate is decremented in the same transaction.

## Privacy

- **R-PRIV-1:** The public and other customers see only an account's **display name** (e.g. on reviews).
- **R-PRIV-2:** An owner sees **full customer info** (display name + email) for accounts that have interacted with their restaurant, meaning they reserved or reviewed. This is shown in the owner's reservation table and on reviews of their restaurant.
- **R-PRIV-3:** Admins see every account's email (admin pages, a restaurant's bookings).

## Profiles

- **R-PROFILE-1:** Every account has a public profile at `/users/{id}`: display name, when they joined, their restaurants and their reviews, with counts. Never the email (R-PRIV-1). Names on reviews, restaurant pages and the owner's booking table link to it.
- **R-PROFILE-2:** Hidden content follows R-ADMIN-3 and -4: the public sees only visible restaurants and reviews on visible restaurants. The profile's owner and admins see everything, marked hidden. A banned account's profile is 404 for everyone but admins.

## Admin

- **R-ADMIN-1:** Admin rights are granted in the database only (`accounts.is_admin`, set with `make admin EMAIL=…`). The app has no screen to grant them. Every `/api/admin` endpoint needs an admin session (401 when logged out, 403 `admin_only` otherwise).
- **R-ADMIN-2:** Admins can list users and restaurants (paged, searchable, filterable by active or banned), and see which restaurants a user owns.
- **R-ADMIN-3 (ban a user):** The account can't log in with a password or Google (403 `account_banned`), and its sessions end at once. Its restaurants are hidden (R-ADMIN-4 applies to them). Its reviews are hidden and removed from every rating they counted in: `rating_sum` and `review_count` are recomputed in the same transaction. Its bookings are left as they are.
- **R-ADMIN-4 (hidden restaurant):** A restaurant is hidden when it is banned or its owner is. Hidden restaurants aren't listed, their page and availability return 404, and they can't be booked or reviewed. Existing bookings stay and can still be cancelled. The owner and admins can still open the page, which shows why it's hidden. Hidden restaurants and banned reviewers don't count toward the Bayesian mean C (ADR-0004).
- **R-ADMIN-5:** Bans are reversible. Unbanning clears `banned_at` and recomputes the ratings, so everything returns exactly as it was. An admin can't ban themselves or another admin (409 `R-ADMIN-5`); remove admin rights in the database first. An optional reason (max 500 characters) is shown to the owner and to admins.
- **R-ADMIN-6 (manage):** Admins can rename any account (not its email, which is the login) and can edit, delete and see the bookings of any restaurant, with the same rules the owner has.
- **R-ADMIN-7 (impersonate):** An admin can use the app as another user, to see what they see and act for them. The admin's session is replaced by an impersonation session for the user that records the admin (8 hours). While it lasts the app shows a banner with a way back, the user can't change their password through it, and admin pages are closed (the session is the user's). "Back" ends it and logs the admin in again. Admins and banned users can't be impersonated (409 `R-ADMIN-7`). The session stops working as soon as the admin loses admin rights or is banned. Every start and stop is logged. See [ADR-0014](decisions/0014-admin-impersonation.md).
