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

// seedBulk adds n accounts, n restaurants (one photo each, open 10:00 to 22:00)
// and about 5n reviews, to check the app with realistic volume
// (docs/backend.md#scaling). Rows are generated in SQL, so 10,000 take seconds.
func seedBulk(ctx context.Context, db *database.DB, images storage.Store, n int) error {
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
				SELECT format('bulk%s@example.com', g), format('Guest %s', g) FROM generate_series(1, $1) g`, []any{n}},
			{"passwords", `
				INSERT INTO auth_identities (account_id, provider, provider_subject, password_hash)
				SELECT id, 'password', id::text, $2 FROM accounts WHERE email LIKE $1`, []any{bulkEmail, string(hash)}},
			// One restaurant per bulk account, with a generated name, cuisine and size.
			{"restaurants", `
				INSERT INTO restaurants (owner_id, name, description, cuisine, location, seats, timezone)
				SELECT a.id,
					(ARRAY['Baan','Krua','Ran','Jay','Mae','Pa','Lung','Sabai','Aroi','Talad'])[1 + (a.id % 10)]
						|| ' ' || (ARRAY['Siam','Riverside','Corner','Garden','Night','Market','Old Town','Canal','Temple','Station'])[1 + (a.id / 10 % 10)]
						|| ' ' || a.id,
					'Generated for load testing.',
					(ARRAY['Thai','Isan','Noodles','BBQ','Café','Seafood','Japanese','Chinese','Vegetarian','Dessert'])[1 + floor(random() * 10)::int],
					(ARRAY['Ari','Silom','Thonglor','Ratchada','Bang Rak','Chatuchak','Sathorn','Ekkamai'])[1 + floor(random() * 8)::int] || ', Bangkok',
					(ARRAY[8, 12, 16, 20, 30, 40])[1 + floor(random() * 6)::int],
					'Asia/Bangkok'
				FROM accounts a WHERE a.email LIKE $1`, []any{bulkEmail}},
			{"hours", `
				INSERT INTO restaurant_hours (restaurant_id, weekday, open_time, close_time)
				SELECT r.id, d, '10:00', '22:00'
				FROM restaurants r JOIN accounts a ON a.id = r.owner_id AND a.email LIKE $1
				CROSS JOIN generate_series(0, 6) d`, []any{bulkEmail}},
			// Each bulk user reviews 5 different restaurants; ratings lean positive like real ones.
			{"reviews", `
				WITH people AS (
					SELECT id, row_number() OVER (ORDER BY id) - 1 AS i FROM accounts WHERE email LIKE $1
				), places AS (
					SELECT r.id, row_number() OVER (ORDER BY r.id) - 1 AS i
					FROM restaurants r JOIN accounts a ON a.id = r.owner_id AND a.email LIKE $1
				)
				INSERT INTO reviews (restaurant_id, account_id, rating, body)
				SELECT pl.id, p.id,
					(ARRAY[2, 3, 4, 4, 4, 5, 5, 5, 5, 3])[1 + floor(random() * 10)::int],
					'Generated review.'
				FROM people p CROSS JOIN generate_series(1, 5) j
				JOIN places pl ON pl.i = (p.i * 7 + j * 131) % $2
				WHERE pl.id <> p.id
				ON CONFLICT DO NOTHING`, []any{bulkEmail, n}},
			{"rating totals", `
				UPDATE restaurants r SET rating_sum = s.sum, review_count = s.n
				FROM (SELECT restaurant_id, sum(rating) AS sum, count(*) AS n FROM reviews GROUP BY restaurant_id) s
				WHERE s.restaurant_id = r.id`, nil},
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
	fmt.Printf("bulk seed of %d done in %s (password %q for bulkN@example.com)\n", n, time.Since(start).Round(time.Millisecond), password)
	return nil
}

// removeBulk deletes every generated account (restaurants, bookings and
// reviews cascade), their photos, and fixes the rating totals of the rest.
func removeBulk(ctx context.Context, db *database.DB, images storage.Store) error {
	rows, _ := db.Pool.Query(ctx, `
		SELECT i.object_key FROM restaurant_images i
		JOIN restaurants r ON r.id = i.restaurant_id JOIN accounts a ON a.id = r.owner_id
		WHERE a.email LIKE $1`, bulkEmail)
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	var removed int64
	err = db.WithinTx(ctx, func(ctx context.Context) error {
		q := db.Conn(ctx)
		tag, err := q.Exec(ctx, `DELETE FROM accounts WHERE email LIKE $1`, bulkEmail)
		if err != nil {
			return err
		}
		removed = tag.RowsAffected()
		_, err = q.Exec(ctx, `
			UPDATE restaurants r SET rating_sum = coalesce(s.sum, 0), review_count = coalesce(s.n, 0)
			FROM (SELECT r2.id, sum(rv.rating) AS sum, count(rv.id) AS n
			      FROM restaurants r2 LEFT JOIN reviews rv ON rv.restaurant_id = r2.id GROUP BY r2.id) s
			WHERE s.id = r.id`)
		return err
	})
	if err != nil {
		return err
	}
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
