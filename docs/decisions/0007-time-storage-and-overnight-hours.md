# 0007 — UTC everywhere, browser-local display, overnight and 24h shifts

**Status:** Accepted (2026-09-26)

## Context
Restaurants can open past midnight (18:00–02:00) or around the clock, and the interview asks how we store time and handle timezones.

## Decision
- **The backend runs in UTC.** Reservations store `starts_at` and `ends_at` as UTC `timestamptz`, and the API exchanges UTC ISO-8601 only. A reservation is an instant range, so crossing midnight needs no special case.
- **The frontend auto-detects the viewer's timezone** (`Intl.DateTimeFormat().resolvedOptions().timeZone`) and converts for display and input. There is no timezone picker.
- **Opening hours** are weekly wall-clock times, and "Monday 18:00" only means something in a specific place. So each restaurant stores a hidden IANA `timezone`, **captured from the owner's browser at creation**. It is used only to expand weekly hours into UTC instants (R-TIME-5).
- **Shift rules:** `close < open` means the shift closes the next day, and `close == open` means open 24h (R-HOURS-3).
- **Validation (R-HOURS-4):** expand the shifts for the start date and the day before into UTC instants with `time.LoadLocation`, merge the ones that touch or overlap, and require `[start, end)` to fit inside one merged period. With merging, bookings that span two back-to-back shifts (24h restaurants) work without special cases.
- The 15-minute grid is checked on UTC instants. Every real-world UTC offset is a multiple of 15 minutes, so a slot is a slot in every timezone.
- The Go binary embeds tz data (`import _ "time/tzdata"`) so slim or distroless images work.

## Consequences
- A viewer in another timezone sees a Bangkok restaurant's hours converted to their own local time. That is correct for instants but may look odd ("opens 12:00" instead of 18:00). At exam scale everyone is in Asia/Bangkok, so this is accepted.
- Thailand has no DST, but `LoadLocation` keeps the expansion correct in timezones that do.
