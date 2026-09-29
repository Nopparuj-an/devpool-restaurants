package review_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"restaurants/internal/testutil/apitest"
)

// ADR-0015: rating totals are kept by triggers, so they stay right whatever
// changes the reviews, including SQL run by hand.

type totals struct{ sum, n int }

func dbTotals(t *testing.T, db *pgxpool.Pool, id int64) totals {
	t.Helper()
	var tt totals
	if err := db.QueryRow(context.Background(),
		`SELECT rating_sum, review_count FROM restaurants WHERE id = $1`, id).Scan(&tt.sum, &tt.n); err != nil {
		t.Fatal(err)
	}
	return tt
}

func exec(t *testing.T, db *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := db.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func accountID(t *testing.T, db *pgxpool.Pool, email string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(context.Background(), `SELECT id FROM accounts WHERE email = $1`, email).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestTotalsFollowSQLChanges(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Rated", 10))
	other := owner.CreateRestaurant(apitest.RestaurantInput("Other", 10))
	for email, stars := range map[string]int{"alice@example.com": 5, "bob@example.com": 3, "carol@example.com": 4} {
		c := env.Signup(email, "R")
		c.Do("PUT", fmt.Sprintf("/api/restaurants/%d/reviews/me", id), review(stars, "ok")).Expect(http.StatusCreated)
		c.Do("PUT", fmt.Sprintf("/api/restaurants/%d/reviews/me", other), review(stars, "ok")).Expect(http.StatusCreated)
	}
	if got := dbTotals(t, env.DB, id); got != (totals{12, 3}) {
		t.Fatalf("after reviews: %+v", got)
	}

	// Deleting an account cascades to its reviews; both restaurants drop them.
	exec(t, env.DB, `DELETE FROM accounts WHERE email = 'alice@example.com'`)
	if got := dbTotals(t, env.DB, id); got != (totals{7, 2}) {
		t.Fatalf("after deleting Alice: %+v", got)
	}
	if got := dbTotals(t, env.DB, other); got != (totals{7, 2}) {
		t.Fatalf("other restaurant after deleting Alice: %+v", got)
	}

	exec(t, env.DB, `UPDATE reviews SET rating = 1 WHERE account_id = $1 AND restaurant_id = $2`, accountID(t, env.DB, "bob@example.com"), id)
	if got := dbTotals(t, env.DB, id); got != (totals{5, 2}) {
		t.Fatalf("after changing Bob's rating: %+v", got)
	}
	exec(t, env.DB, `UPDATE reviews SET body = 'edited'`) // text only: totals unchanged
	exec(t, env.DB, `DELETE FROM reviews WHERE restaurant_id = $1`, id)
	if got := dbTotals(t, env.DB, id); got != (totals{0, 0}) {
		t.Fatalf("after deleting all reviews: %+v", got)
	}
	if got := dbTotals(t, env.DB, other); got != (totals{7, 2}) {
		t.Fatalf("other restaurant should be untouched: %+v", got)
	}
}

func TestTotalsRepairAndGuard(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Rated", 10))
	path := fmt.Sprintf("/api/restaurants/%d/reviews/me", id)
	env.Signup("alice@example.com", "Alice").Do("PUT", path, review(5, "ok")).Expect(http.StatusCreated)

	// Impossible totals are rejected outright.
	for _, sql := range []string{
		`UPDATE restaurants SET rating_sum = -1 WHERE id = $1`,
		`UPDATE restaurants SET review_count = -1, rating_sum = 0 WHERE id = $1`,
		`UPDATE restaurants SET rating_sum = 6 WHERE id = $1`, // 1 review can't sum to 6
	} {
		if _, err := env.DB.Exec(context.Background(), sql, id); err == nil {
			t.Errorf("%s: want check violation", sql)
		}
	}

	// Plausible but wrong totals heal on the next review change...
	exec(t, env.DB, `UPDATE restaurants SET rating_sum = 0, review_count = 0 WHERE id = $1`, id)
	env.Signup("bob@example.com", "Bob").Do("PUT", path, review(3, "ok")).Expect(http.StatusCreated)
	if got := dbTotals(t, env.DB, id); got != (totals{8, 2}) {
		t.Fatalf("after next review: %+v", got)
	}
	// ...or at once with `make ratings-recompute`.
	exec(t, env.DB, `UPDATE restaurants SET rating_sum = 0, review_count = 0 WHERE id = $1`, id)
	exec(t, env.DB, `SELECT recompute_ratings(ARRAY(SELECT id FROM restaurants))`)
	if got := dbTotals(t, env.DB, id); got != (totals{8, 2}) {
		t.Fatalf("after recompute: %+v", got)
	}
}

// Two writers review the same restaurant at once, outside the app (which
// would serialize them itself). The second one's recompute must wait for the
// first to commit and then count both reviews. Without the row lock in
// recompute_ratings, it aggregates from a snapshot that misses the first
// review and the count ends at 1.
func TestTotalsConcurrentWriters(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Busy", 10))
	env.Signup("alice@example.com", "Alice")
	env.Signup("bob@example.com", "Bob")
	alice, bob := accountID(t, env.DB, "alice@example.com"), accountID(t, env.DB, "bob@example.com")
	ctx := context.Background()
	const insert = `INSERT INTO reviews (restaurant_id, account_id, rating, body) VALUES ($1, $2, $3, 'ok')`

	first, err := env.DB.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Rollback(ctx)
	if _, err := first.Exec(ctx, insert, id, alice, 5); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := env.DB.Exec(ctx, insert, id, bob, 3)
		done <- err
	}()
	// Commit only once the second writer is blocked on the lock.
	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting bool
		if err := env.DB.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock')`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("second writer never waited for the first")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := first.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := dbTotals(t, env.DB, id); got != (totals{8, 2}) {
		t.Fatalf("after two concurrent reviews: %+v", got)
	}
}
