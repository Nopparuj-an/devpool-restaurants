package reservation_test

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	restaurantrepo "restaurants/internal/restaurant/repository"
	"restaurants/internal/testutil/apitest"
)

var bkk, _ = time.LoadLocation("Asia/Bangkok")

// at is hh:mm Bangkok time, `days` from today.
func at(days int, hhmm string) time.Time {
	var h, m int
	fmt.Sscanf(hhmm, "%d:%d", &h, &m)
	y, mo, d := time.Now().In(bkk).Date()
	return time.Date(y, mo, d+days, h, m, 0, 0, bkk)
}

// q formats t for a query string ("+07:00" must be escaped).
func q(t time.Time) string { return url.QueryEscape(t.Format(time.RFC3339)) }

func book(pax int, from, to time.Time) map[string]any {
	return map[string]any{"pax": pax, "starts_at": from, "ends_at": to}
}

type reservation struct {
	ID              int64     `json:"id"`
	Pax             int       `json:"pax"`
	State           string    `json:"state"`
	CanModify       bool      `json:"can_modify"`
	ModifiableUntil time.Time `json:"modifiable_until"`
	Customer        *struct {
		Email string `json:"email"`
	} `json:"customer"`
}

func create(c *apitest.Client, restaurantID int64, body map[string]any) *apitest.Response {
	return c.Do("POST", fmt.Sprintf("/api/restaurants/%d/reservations", restaurantID), body)
}

func mustCreate(t *testing.T, c *apitest.Client, restaurantID int64, body map[string]any) int64 {
	t.Helper()
	var r reservation
	create(c, restaurantID, body).Expect(http.StatusCreated).JSON(&r)
	return r.ID
}

// The two seat-rule examples from the exam brief, end to end.
func TestBriefCapacityExamples(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	cust := env.Signup("cust@example.com", "Cust")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Klong", 10))

	// 7 booked 12:00–13:00; asking 5 for 12:30–13:00 would make 12 > 10.
	mustCreate(t, cust, id, book(7, at(2, "12:00"), at(2, "13:00")))
	create(cust, id, book(5, at(2, "12:30"), at(2, "13:00"))).ExpectError(http.StatusConflict, "R-BOOK-5")

	// A=7 at 12:00–12:30, B=7 at 12:30–13:00: C=3 for the whole hour fits (peak 10, not 17).
	mustCreate(t, cust, id, book(7, at(3, "12:00"), at(3, "12:30")))
	mustCreate(t, cust, id, book(7, at(3, "12:30"), at(3, "13:00")))
	mustCreate(t, cust, id, book(3, at(3, "12:00"), at(3, "13:00")))
	create(cust, id, book(1, at(3, "12:45"), at(3, "13:00"))).ExpectError(http.StatusConflict, "R-BOOK-5")
	// Back-to-back is not overlap: 13:00 is free again.
	mustCreate(t, cust, id, book(10, at(3, "13:00"), at(3, "14:00")))
}

func TestCreateValidation(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	id := owner.CreateRestaurant(apitest.RestaurantInput("V", 10))

	create(env.Client(), id, book(2, at(2, "12:00"), at(2, "13:00"))).ExpectError(http.StatusUnauthorized, "unauthorized")
	create(owner, 999999, book(2, at(2, "12:00"), at(2, "13:00"))).ExpectError(http.StatusNotFound, "not_found")
	create(owner, id, map[string]any{"pax": 2}).ExpectError(http.StatusUnprocessableEntity, "invalid_input")

	for _, tc := range []struct {
		name string
		body map[string]any
		code string
	}{
		{"pax over seats", book(11, at(2, "12:00"), at(2, "13:00")), "R-BOOK-1"},
		{"off grid", book(2, at(2, "12:05"), at(2, "13:00")), "R-BOOK-2"},
		{"too long", book(2, at(2, "12:00"), at(2, "16:15")), "R-BOOK-2"},
		{"past", book(2, at(-1, "12:00"), at(-1, "13:00")), "R-BOOK-3"},
		{"outside hours", book(2, at(2, "22:30"), at(2, "23:30")), "R-BOOK-4"},
		{"beyond horizon", book(2, at(32, "12:00"), at(32, "13:00")), "R-BOOK-8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			create(owner, id, tc.body).ExpectError(http.StatusUnprocessableEntity, tc.code)
		})
	}

	// R-BOOK-7: owners may book their own restaurant.
	mustCreate(t, owner, id, book(2, at(2, "12:00"), at(2, "13:00")))
}

// Opening hours are interpreted in the restaurant's timezone (R-TIME-5).
func TestRestaurantTimezone(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	in := apitest.RestaurantInput("Tokyo", 10)
	in["timezone"] = "Asia/Tokyo" // UTC+9, two hours ahead of Bangkok
	hours := make([]map[string]any, 7)
	for d := range hours {
		hours[d] = map[string]any{"weekday": d, "open": "10:00", "close": "12:00"}
	}
	in["hours"] = hours
	id := owner.CreateRestaurant(in)

	mustCreate(t, owner, id, book(2, at(2, "08:00"), at(2, "10:00"))) // 10:00–12:00 in Tokyo
	create(owner, id, book(2, at(2, "10:00"), at(2, "11:00"))).ExpectError(http.StatusUnprocessableEntity, "R-BOOK-4")
}

func TestOvernightBooking(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	in := apitest.RestaurantInput("Late", 10)
	hours := make([]map[string]any, 7)
	for d := range hours {
		hours[d] = map[string]any{"weekday": d, "open": "18:00", "close": "02:00"}
	}
	in["hours"] = hours
	id := owner.CreateRestaurant(in)

	mustCreate(t, owner, id, book(2, at(2, "23:00"), at(3, "01:00")))
	mustCreate(t, owner, id, book(2, at(3, "01:00"), at(3, "02:00")))
	create(owner, id, book(2, at(3, "01:30"), at(3, "02:30"))).ExpectError(http.StatusUnprocessableEntity, "R-BOOK-4")
}

func TestEdit(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	alice := env.Signup("alice@example.com", "Alice")
	bob := env.Signup("bob@example.com", "Bob")
	id := owner.CreateRestaurant(apitest.RestaurantInput("E", 10))

	// Brief: 10 seats, A=7 and B=3 in the same slot (exactly full).
	a := mustCreate(t, alice, id, book(7, at(2, "12:00"), at(2, "13:00")))
	mustCreate(t, bob, id, book(3, at(2, "12:00"), at(2, "13:00")))
	path := fmt.Sprintf("/api/reservations/%d", a)

	alice.Do("PUT", path, book(8, at(2, "12:00"), at(2, "13:00"))).ExpectError(http.StatusConflict, "R-BOOK-5")
	var r reservation
	alice.Do("PUT", path, book(5, at(2, "12:00"), at(2, "13:00"))).Expect(http.StatusOK).JSON(&r)
	if r.Pax != 5 {
		t.Errorf("pax = %d, want 5", r.Pax)
	}
	// The freed seats are really free: 2 more fit.
	mustCreate(t, bob, id, book(2, at(2, "12:30"), at(2, "13:00")))

	bob.Do("PUT", path, book(1, at(2, "12:00"), at(2, "13:00"))).ExpectError(http.StatusForbidden, "forbidden")
	bob.Do("GET", path, nil).ExpectError(http.StatusNotFound, "not_found")
	alice.Do("PUT", "/api/reservations/999999", book(1, at(2, "12:00"), at(2, "13:00"))).ExpectError(http.StatusNotFound, "not_found")

	alice.Do("POST", path+"/cancel", nil).Expect(http.StatusOK)
	alice.Do("PUT", path, book(1, at(2, "12:00"), at(2, "13:00"))).ExpectError(http.StatusConflict, "not_active")
	alice.Do("POST", path+"/cancel", nil).ExpectError(http.StatusConflict, "not_active")
}

// R-EDIT-3: after the owner lowers seats, customers can still shrink bookings.
func TestShrinkAfterSeatsLowered(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	alice := env.Signup("alice@example.com", "Alice")
	id := owner.CreateRestaurant(apitest.RestaurantInput("S", 10))

	a := mustCreate(t, alice, id, book(4, at(2, "12:00"), at(2, "13:00")))
	mustCreate(t, alice, id, book(4, at(2, "12:00"), at(2, "13:00")))
	owner.Do("PUT", fmt.Sprintf("/api/restaurants/%d", id), apitest.RestaurantInput("S", 6)).Expect(http.StatusOK)

	path := fmt.Sprintf("/api/reservations/%d", a)
	alice.Do("PUT", path, book(3, at(2, "12:15"), at(2, "13:00"))).Expect(http.StatusOK) // 4+3 > 6, but only shrinks
	alice.Do("PUT", path, book(3, at(2, "12:00"), at(2, "13:00"))).ExpectError(http.StatusConflict, "R-BOOK-5")
}

func TestCancelCutoff(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	in := apitest.RestaurantInput("24h", 10)
	hours := make([]map[string]any, 7)
	for d := range hours {
		hours[d] = map[string]any{"weekday": d, "open": "00:00", "close": "00:00"}
	}
	in["hours"] = hours
	id := owner.CreateRestaurant(in)

	// Starts in 15–30 minutes: bookable, but already inside the 30-minute cutoff.
	soon := time.Now().Truncate(15 * time.Minute).Add(30 * time.Minute)
	near := mustCreate(t, owner, id, book(2, soon, soon.Add(time.Hour)))
	var r reservation
	owner.Do("GET", fmt.Sprintf("/api/reservations/%d", near), nil).Expect(http.StatusOK).JSON(&r)
	if r.CanModify || r.State != "upcoming" {
		t.Errorf("near reservation: can_modify=%v state=%s", r.CanModify, r.State)
	}
	owner.Do("POST", fmt.Sprintf("/api/reservations/%d/cancel", near), nil).ExpectError(http.StatusConflict, "R-CANCEL-1")
	owner.Do("PUT", fmt.Sprintf("/api/reservations/%d", near), book(1, soon, soon.Add(time.Hour))).
		ExpectError(http.StatusConflict, "R-EDIT-1")

	// Far enough ahead: cancel works and frees the seats.
	far := mustCreate(t, owner, id, book(10, at(2, "12:00"), at(2, "13:00")))
	owner.Do("POST", fmt.Sprintf("/api/reservations/%d/cancel", far), nil).Expect(http.StatusOK).JSON(&r)
	if r.State != "cancelled" || r.CanModify {
		t.Errorf("after cancel: state=%s can_modify=%v", r.State, r.CanModify)
	}
	mustCreate(t, owner, id, book(10, at(2, "12:00"), at(2, "13:00")))
}

// R-BOOK-6 / ADR-0003, deterministic version: while another transaction
// holds the restaurant lock, a booking must wait, then see that
// transaction's committed reservation and fail. Without FOR UPDATE the
// booking would not wait and would overbook (verified by removing it).
func TestBookingWaitsForRestaurantLock(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	cust := env.Signup("cust@example.com", "Cust")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Lock", 6))
	var me struct{ ID int64 }
	owner.Do("GET", "/api/me", nil).Expect(http.StatusOK).JSON(&me)
	from, to := at(2, "19:00"), at(2, "20:00")

	ctx := t.Context()
	tx, err := env.DB.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, _, err := restaurantrepo.LoadRules(ctx, tx, id, true); err != nil {
		t.Fatal(err)
	}

	done := make(chan *apitest.Response, 1)
	go func() { done <- create(cust, id, book(6, from, to)) }()
	select {
	case r := <-done:
		t.Fatalf("booking finished (%d) while the restaurant was locked", r.Status)
	case <-time.After(300 * time.Millisecond):
	}

	// The lock holder takes all seats and commits; the waiting booking resumes.
	if _, err := tx.Exec(ctx, `INSERT INTO reservations (restaurant_id, account_id, pax, starts_at, ends_at)
		VALUES ($1, $2, 6, $3, $4)`, id, me.ID, from, to); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	(<-done).ExpectError(http.StatusConflict, "R-BOOK-5")
}

// R-BOOK-6 smoke test: many customers race for the last seats; exactly one
// wins. (Timing-dependent, so TestBookingWaitsForRestaurantLock is the real proof.)
func TestConcurrentLastSeats(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Race", 6))

	const racers = 12
	clients := make([]*apitest.Client, racers)
	for i := range clients {
		clients[i] = env.Signup(fmt.Sprintf("racer%d@example.com", i), "Racer")
	}

	var (
		wg       sync.WaitGroup
		start    = make(chan struct{})
		statuses = make(chan int, racers)
	)
	for _, c := range clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			statuses <- create(c, id, book(6, at(2, "19:00"), at(2, "20:00"))).Status
		}()
	}
	close(start)
	wg.Wait()
	close(statuses)

	count := map[int]int{}
	for s := range statuses {
		count[s]++
	}
	if count[http.StatusCreated] != 1 || count[http.StatusConflict] != racers-1 {
		t.Fatalf("statuses = %v, want exactly 1×201 and %d×409", count, racers-1)
	}
	var total int
	env.DB.QueryRow(t.Context(), `SELECT coalesce(sum(pax), 0) FROM reservations WHERE status = 'active'`).Scan(&total)
	if total != 6 {
		t.Fatalf("booked pax = %d, want 6", total)
	}
}

func TestListsAndPrivacy(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	alice := env.Signup("alice@example.com", "Alice")
	id := owner.CreateRestaurant(apitest.RestaurantInput("L", 10))

	later := mustCreate(t, alice, id, book(2, at(3, "12:00"), at(3, "13:00")))
	sooner := mustCreate(t, alice, id, book(2, at(2, "12:00"), at(2, "13:00")))

	var mine struct{ Reservations []reservation }
	alice.Do("GET", "/api/me/reservations", nil).Expect(http.StatusOK).JSON(&mine)
	if len(mine.Reservations) != 2 || mine.Reservations[0].ID != sooner || mine.Reservations[1].ID != later {
		t.Fatalf("mine order = %+v, want sooner then later", mine.Reservations)
	}
	if mine.Reservations[0].Customer != nil {
		t.Error("customer details leaked into my own list")
	}

	path := fmt.Sprintf("/api/restaurants/%d/reservations?from=%s&to=%s", id,
		q(at(0, "00:00")), q(at(7, "00:00")))
	var table struct{ Reservations []reservation }
	owner.Do("GET", path, nil).Expect(http.StatusOK).JSON(&table)
	if len(table.Reservations) != 2 || table.Reservations[0].Customer == nil ||
		table.Reservations[0].Customer.Email != "alice@example.com" {
		t.Fatalf("owner table = %+v", table.Reservations)
	}
	alice.Do("GET", path, nil).ExpectError(http.StatusForbidden, "forbidden")
}

func TestAvailability(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	id := owner.CreateRestaurant(apitest.RestaurantInput("A", 10))
	mustCreate(t, owner, id, book(8, at(2, "12:00"), at(2, "12:30")))

	var a struct {
		LimitedThreshold int  `json:"limited_threshold"`
		Limited          bool `json:"limited"`
		Slots            []struct {
			Start     time.Time `json:"start"`
			SeatsLeft int       `json:"seats_left"`
			Limited   bool      `json:"limited"`
		} `json:"slots"`
	}
	path := fmt.Sprintf("/api/restaurants/%d/availability?from=%s&to=%s", id,
		q(at(2, "09:30")), q(at(2, "13:00")))
	env.Client().Do("GET", path, nil).Expect(http.StatusOK).JSON(&a)

	// Opens at 10:00, so slots are 10:00 … 12:45 (12 slots).
	if len(a.Slots) != 12 || !a.Slots[0].Start.Equal(at(2, "10:00")) {
		t.Fatalf("slots = %+v", a.Slots)
	}
	left := map[string]int{}
	for _, s := range a.Slots {
		left[s.Start.In(bkk).Format("15:04")] = s.SeatsLeft
	}
	if left["11:45"] != 10 || left["12:00"] != 2 || left["12:15"] != 2 || left["12:30"] != 10 {
		t.Errorf("seats left = %v", left)
	}
	if a.LimitedThreshold != 2 || !a.Limited {
		t.Errorf("threshold = %d limited = %v, want 2 true", a.LimitedThreshold, a.Limited)
	}
}
