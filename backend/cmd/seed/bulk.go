package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"restaurants/internal/platform/database"
	"restaurants/internal/platform/storage"
)

// bulkEmail marks generated accounts so they can be removed again.
const bulkEmail = "bulk%@example.com"

// seedBulk adds n test accounts, n test restaurants (one photo each, open
// 10:00 to 22:00), about 5n reviews and 5n reservations, to check paging and
// query speed with realistic volume (docs/backend.md#scaling). Rows are
// generated in SQL, so 10,000 take seconds.
//
// Everything is named "Test …" so nobody mistakes it for real content. Two
// rows are made deliberately heavy, so every paged list has several pages:
// "Test User 1" writes up to 150 reviews and bookings, and "Test Restaurant 2"
// gets up to 150 reviews and a booking in two slots on every day.
//
// Bookings obey the rules on purpose: 15-minute grid, inside opening hours,
// future ones within 30 days (R-BOOK-2/4/8). At most three bookings overlap
// at one restaurant and each takes at most a quarter of its seats, so the
// capacity rule holds (R-BOOK-5). Nobody reviews their own restaurant (R-REVIEW-3).
func seedBulk(ctx context.Context, db *database.DB, images storage.Store, n int) error {
	if n < 10 {
		return fmt.Errorf("-bulk needs N >= 10")
	}
	var exists bool
	if err := db.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM accounts WHERE email LIKE $1)`, bulkEmail).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("bulk data already present; run with -bulk-remove first")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	start := time.Now()
	heavy := min(150, n-2)

	// people and places number the bulk accounts and their restaurants from 0,
	// in the same order, so person i owns place i.
	const people = `
		people AS (
			SELECT id, display_name AS name, row_number() OVER (ORDER BY id) - 1 AS i
			FROM accounts WHERE email LIKE $1
		)`
	const numbered = people + `, places AS (
			SELECT r.id, r.seats, row_number() OVER (ORDER BY r.id) - 1 AS i
			FROM restaurants r JOIN accounts a ON a.id = r.owner_id AND a.email LIKE $1
		)`
	// A booking day in Bangkok, d days from today, as a timestamptz at hh:mm.
	const at = `((((now() AT TIME ZONE 'Asia/Bangkok')::date + (%[1]s)::int) + %[2]s::time) AT TIME ZONE 'Asia/Bangkok')`
	bookingCols := `restaurant_id, account_id, pax, starts_at, ends_at, status, created_at, updated_at`
	bookingVals := `
		SELECT rid, aid,
			1 + floor(random() * greatest(1, seats / 4))::int,
			s, s + dur * interval '1 minute',
			CASE WHEN random() < 0.1 THEN 'cancelled' ELSE 'active' END,
			least(now(), s - interval '2 days'), least(now(), s - interval '2 days')
		FROM picks`

	var ids []int64
	err = db.WithinTx(ctx, func(ctx context.Context) error {
		q := db.Conn(ctx)
		if _, err := q.Exec(ctx, `SELECT setseed(0.42)`); err != nil {
			return err
		}
		steps := []struct {
			name string
			sql  string
			args []any
		}{
			{"accounts", `
				INSERT INTO accounts (email, display_name)
				SELECT format('bulk%s@example.com', g),
					CASE g WHEN 1 THEN 'Test User 1 (many reviews and bookings)' ELSE format('Test User %s', g) END
				FROM generate_series(1, $1) g`, []any{n}},
			{"passwords", `
				INSERT INTO auth_identities (account_id, provider, provider_subject, password_hash)
				SELECT id, 'password', id::text, $2 FROM accounts WHERE email LIKE $1`, []any{bulkEmail, string(hash)}},
			// One restaurant per bulk account. The cuisine is in the name so search has something to find.
			{"restaurants", `
				WITH ` + people + `
				INSERT INTO restaurants (owner_id, name, description, cuisine, location, seats, timezone)
				SELECT p.id,
					CASE p.i WHEN 1 THEN 'Test Restaurant 2 (many reviews and bookings)'
						ELSE format('Test Restaurant %s · %s', p.i + 1, c.cuisine) END,
					'Generated test data for load and paging checks. Not a real restaurant.',
					c.cuisine,
					format('Test Area %s', 1 + p.i % 8),
					CASE p.i WHEN 1 THEN 40 ELSE (ARRAY[8, 12, 16, 20, 30, 40])[1 + floor(random() * 6)::int] END,
					'Asia/Bangkok'
				FROM people p
				CROSS JOIN LATERAL (SELECT (ARRAY['Thai','Isan','Noodles','BBQ','Café','Seafood','Japanese','Chinese','Vegetarian','Dessert'])[1 + (p.i * 7 + p.id) % 10] AS cuisine) c
				ORDER BY p.id`, []any{bulkEmail}},
			{"hours", `
				INSERT INTO restaurant_hours (restaurant_id, weekday, open_time, close_time)
				SELECT r.id, d, '10:00', '22:00'
				FROM restaurants r JOIN accounts a ON a.id = r.owner_id AND a.email LIKE $1
				CROSS JOIN generate_series(0, 6) d`, []any{bulkEmail}},
			// Each test user reviews 5 other restaurants; ratings lean positive like real ones.
			{"reviews", `
				WITH ` + numbered + `, picks AS (
					SELECT DISTINCT ON (pl.id, p.id) pl.id AS rid, p.id AS aid, p.name,
						(ARRAY[2, 3, 4, 4, 4, 5, 5, 5, 5, 3])[1 + floor(random() * 10)::int] AS rating
					FROM people p CROSS JOIN generate_series(1, 5) j
					JOIN places pl ON pl.i = (p.i * 7 + j * 131) % $2
					WHERE pl.i <> p.i
				)
				INSERT INTO reviews (restaurant_id, account_id, rating, body)
				SELECT rid, aid, rating, format('Test review: %s stars from %s. Generated data, not a real opinion.', rating, name)
				FROM picks
				ON CONFLICT DO NOTHING`, []any{bulkEmail, n}},
			// Test User 1 reviews many restaurants, and Test Restaurant 2 gets many reviews.
			{"heavy reviews", `
				WITH ` + numbered + `, picks AS (
					SELECT pl.id AS rid, p.id AS aid, p.name, (ARRAY[3, 4, 4, 5, 5])[1 + floor(random() * 5)::int] AS rating
					FROM people p JOIN places pl ON pl.i BETWEEN 1 AND $2 WHERE p.i = 0
					UNION ALL
					SELECT pl.id, p.id, p.name, (ARRAY[2, 3, 4, 4, 5])[1 + floor(random() * 5)::int]
					FROM places pl JOIN people p ON p.i BETWEEN 2 AND $2 + 1 WHERE pl.i = 1
				)
				INSERT INTO reviews (restaurant_id, account_id, rating, body)
				SELECT rid, aid, rating, format('Test review: %s stars from %s. Generated data, not a real opinion.', rating, name)
				FROM picks
				ON CONFLICT DO NOTHING`, []any{bulkEmail, heavy}},
			// 5 bookings per restaurant, 2 past and 3 upcoming, each on its own day.
			{"reservations", `
				WITH ` + numbered + `, slots (k, d, t, dur) AS (
					VALUES (1, -20, '11:00', 90), (2, -6, '12:00', 60), (3, 2, '18:00', 90), (4, 8, '19:00', 120), (5, 15, '20:00', 120)
				), picks AS (
					SELECT pl.id AS rid, p.id AS aid, pl.seats, sl.dur, ` + fmt.Sprintf(at, "sl.d", "sl.t") + ` AS s
					FROM places pl CROSS JOIN slots sl
					JOIN people p ON p.i = (pl.i + sl.k * 37) % $2
				)
				INSERT INTO reservations (` + bookingCols + `)` + bookingVals, []any{bulkEmail, n}},
			// Test User 1 books many restaurants at 15:00, a time no other generated booking uses.
			{"heavy bookings", `
				WITH ` + numbered + `, picks AS (
					SELECT pl.id AS rid, p.id AS aid, pl.seats, 60 AS dur,
						` + fmt.Sprintf(at, "(CASE WHEN pl.i % 44 >= 15 THEN pl.i % 44 - 14 ELSE pl.i % 44 - 15 END)", "'15:00'") + ` AS s
					FROM people p JOIN places pl ON pl.i BETWEEN 1 AND $2 WHERE p.i = 0
				)
				INSERT INTO reservations (` + bookingCols + `)` + bookingVals, []any{bulkEmail, heavy}},
			// Test Restaurant 2 has two bookings a day from 60 days ago to 29 days ahead.
			{"heavy calendar", `
				WITH ` + numbered + `, picks AS (
					SELECT pl.id AS rid, p.id AS aid, pl.seats, sl.dur, ` + fmt.Sprintf(at, "d", "sl.t") + ` AS s
					FROM places pl
					CROSS JOIN generate_series(-60, 29) d
					CROSS JOIN (VALUES (0, '11:00', 90), (1, '14:00', 90)) sl (k, t, dur)
					JOIN people p ON p.i = (d * 2 + sl.k + 1000 * $2) % $2
					WHERE pl.i = 1 AND d <> 0
				)
				INSERT INTO reservations (` + bookingCols + `)` + bookingVals, []any{bulkEmail, n}},
		}
		for _, st := range steps {
			tag, err := q.Exec(ctx, st.sql, st.args...)
			if err != nil {
				return fmt.Errorf("%s: %w", st.name, err)
			}
			fmt.Printf("  %-14s %6d rows\n", st.name, tag.RowsAffected())
		}
		rows, _ := q.Query(ctx, `
			SELECT r.id FROM restaurants r JOIN accounts a ON a.id = r.owner_id AND a.email LIKE $1 ORDER BY r.id`, bulkEmail)
		ids, err = pgx.CollectRows(rows, pgx.RowTo[int64])
		return err
	})
	if err != nil {
		return err
	}
	// Fresh planner statistics, so the rating triggers and list queries use
	// their indexes right away instead of after autovacuum gets to it.
	if _, err := db.Pool.Exec(ctx, `ANALYZE`); err != nil {
		return err
	}

	// One small photo per restaurant, uploaded with a few workers.
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		jobs     = make(chan int64)
	)
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				key := fmt.Sprintf("restaurants/%d/bulk-cover.png", id)
				err := images.Put(ctx, key, "image/png", artSized(480, 320, float64(id*37%360), int(id)))
				if err == nil {
					_, err = db.Pool.Exec(ctx, `INSERT INTO restaurant_images (restaurant_id, object_key, is_cover) VALUES ($1, $2, true)`, id, key)
				}
				if err != nil {
					mu.Lock()
					firstErr = cmpErr(firstErr, err)
					mu.Unlock()
				}
			}
		}()
	}
	for _, id := range ids {
		jobs <- id
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return fmt.Errorf("photos: %w", firstErr)
	}
	fmt.Printf("  %-14s %6d objects\n", "photos", len(ids))
	fmt.Printf("bulk seed of %d done in %s\n", n, time.Since(start).Round(time.Millisecond))
	fmt.Printf("  log in as bulkN@example.com (password %q); bulk1 has the most reviews and bookings,\n", password)
	fmt.Printf("  and \"Test Restaurant 2\" (id %d) the most reviews and bookings of any restaurant\n", ids[1])
	return nil
}

// removeBulk deletes every generated account (restaurants, bookings and
// reviews cascade) and their photos. The review triggers fix the rating
// totals of the remaining restaurants (ADR-0015).
func removeBulk(ctx context.Context, db *database.DB, images storage.Store) error {
	rows, _ := db.Pool.Query(ctx, `
		SELECT i.object_key FROM restaurant_images i
		JOIN restaurants r ON r.id = i.restaurant_id JOIN accounts a ON a.id = r.owner_id
		WHERE a.email LIKE $1`, bulkEmail)
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	tag, err := db.Pool.Exec(ctx, `DELETE FROM accounts WHERE email LIKE $1`, bulkEmail)
	if err != nil {
		return err
	}
	removed := tag.RowsAffected()
	for i := 0; i < len(keys); i += 1000 { // S3 deletes at most 1000 keys per call
		if err := images.Delete(ctx, keys[i:min(i+1000, len(keys))]...); err != nil {
			return err
		}
	}
	fmt.Printf("removed %d bulk accounts and %d photos\n", removed, len(keys))
	return nil
}

func cmpErr(first, err error) error {
	if first != nil {
		return first
	}
	return err
}
