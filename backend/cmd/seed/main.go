// Command seed fills an empty database with demo accounts, restaurants,
// images, reservations and reviews, so the app has something to show right
// after setup. It goes through the same services as the API, so every
// business rule applies. Run with `make seed`; it does nothing if already seeded.
//
// `-bulk N` adds N generated users and restaurants for load checks, and
// `-bulk-remove` takes them out again (make seed-bulk / seed-bulk-remove).
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"math"
	"os"
	"time"
	_ "time/tzdata"

	"github.com/jackc/pgx/v5/pgxpool"

	"restaurants/internal/auth"
	authservice "restaurants/internal/auth/service"
	"restaurants/internal/platform/config"
	"restaurants/internal/platform/database"
	"restaurants/internal/platform/storage"
	"restaurants/internal/reservation"
	reservationmodel "restaurants/internal/reservation/model"
	reservationservice "restaurants/internal/reservation/service"
	"restaurants/internal/restaurant"
	restaurantmodel "restaurants/internal/restaurant/model"
	restaurantservice "restaurants/internal/restaurant/service"
	"restaurants/internal/review"
	reviewmodel "restaurants/internal/review/model"
	reviewservice "restaurants/internal/review/service"
)

const password = "password123"

var bkk, _ = time.LoadLocation("Asia/Bangkok")

func main() {
	if err := run(); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}
}

type seeder struct {
	ctx          context.Context
	pool         *pgxpool.Pool
	auth         authservice.Service
	restaurants  restaurantservice.Service
	reservations reservationservice.Service
	reviews      reviewservice.Service
	accounts     map[string]int64
}

func run() error {
	bulk := flag.Int("bulk", 0, "add N generated users and restaurants")
	bulkRemove := flag.Bool("bulk-remove", false, "remove generated users and restaurants")
	flag.Parse()
	time.Local = time.UTC
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool); err != nil {
		return err
	}

	if *bulk > 0 || *bulkRemove {
		db, images := database.New(pool), storage.NewS3(cfg.S3)
		if *bulkRemove {
			return removeBulk(ctx, db, images)
		}
		return seedBulk(ctx, db, images, *bulk)
	}

	var seeded bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM accounts WHERE email = 'somchai@example.com')`).Scan(&seeded); err != nil {
		return err
	}
	if seeded {
		fmt.Println("already seeded, nothing to do (reset: make down && rm -rf deployment/data && make up garage-init seed)")
		return nil
	}

	db := database.New(pool)
	s := &seeder{
		ctx:          ctx,
		pool:         pool,
		auth:         auth.NewService(db),
		restaurants:  restaurant.NewService(db, storage.NewS3(cfg.S3), cfg.ImageBaseURL),
		reservations: reservation.NewService(db, cfg.ImageBaseURL),
		reviews:      review.NewService(db),
		accounts:     map[string]int64{},
	}
	return s.seed()
}

func (s *seeder) seed() error {
	for _, a := range []struct{ email, name string }{
		{"somchai@example.com", "Somchai"}, // owns 3 restaurants
		{"malee@example.com", "Malee"},     // owns 2 restaurants
		{"alice@example.com", "Alice"},
		{"bob@example.com", "Bob"},
		{"carol@example.com", "Carol"},
		{"dan@example.com", "Dan"},
		{"admin@example.com", "Admin"}, // made admin below, in the database (R-ADMIN-1)
	} {
		acc, err := s.auth.Signup(s.ctx, a.email, password, a.name)
		if err != nil {
			return fmt.Errorf("signup %s: %w", a.email, err)
		}
		s.accounts[a.name] = acc.ID
	}
	if _, err := s.pool.Exec(s.ctx, `UPDATE accounts SET is_admin = true WHERE email = 'admin@example.com'`); err != nil {
		return err
	}

	daily := func(open, close string, closedOn ...time.Weekday) []restaurantmodel.Hours {
		var h []restaurantmodel.Hours
	days:
		for d := time.Sunday; d <= time.Saturday; d++ {
			for _, c := range closedOn {
				if d == c {
					continue days
				}
			}
			h = append(h, restaurantmodel.Hours{Weekday: int(d), Open: open, Close: close})
		}
		return h
	}
	type spec struct {
		owner string
		in    restaurantmodel.Input
		hue   float64
	}
	specs := []spec{
		{"Somchai", restaurantmodel.Input{
			Name: "ครัวริมคลอง", Cuisine: "Thai", Location: "Khlong Bang Luang, Bangkok", Seats: 10,
			Description: "Thai home cooking by the canal. The room is small, so book ahead for lunch.",
			Hours:       daily("11:00", "22:00", time.Monday),
		}, 25},
		{"Somchai", restaurantmodel.Input{
			Name: "ส้มตำหน้าตลาด", Cuisine: "Isan", Location: "Talad Noi, Bangkok", Seats: 24,
			Description: "Market-front som tam, grilled chicken and sticky rice.",
			Hours:       daily("10:00", "21:00"),
		}, 95},
		{"Somchai", restaurantmodel.Input{
			Name: "ก๋วยเตี๋ยวเรือหน้าวัด", Cuisine: "Noodles", Location: "Victory Monument, Bangkok", Seats: 16,
			Description: "Boat noodles in small bowls, the way they were served from the canals.",
			Hours:       daily("09:00", "16:00"),
		}, 10},
		{"Malee", restaurantmodel.Input{
			Name: "Midnight Moo Kra Ta", Cuisine: "BBQ", Location: "Ratchada, Bangkok", Seats: 30,
			Description:         "Thai barbecue hotpot that opens at dusk and runs past midnight.",
			Hours:               daily("18:00", "02:00"),
			CancelCutoffMinutes: ptr(120),
		}, 340},
		{"Malee", restaurantmodel.Input{
			Name: "Sabai 24h Café", Cuisine: "Café", Location: "Ari, Bangkok", Seats: 12,
			Description:           "Coffee, toast and khao tom around the clock.",
			Hours:                 daily("00:00", "00:00"),
			MaxReservationMinutes: ptr(120),
		}, 200},
	}
	ids := map[string]int64{}
	for i, sp := range specs {
		sp.in.Timezone = "Asia/Bangkok"
		uploads := []restaurantmodel.Upload{
			{Data: art(sp.hue, i*3), ContentType: "image/png", Ext: ".png"},
			{Data: art(sp.hue+30, i*3+1), ContentType: "image/png", Ext: ".png"},
			{Data: art(sp.hue-30, i*3+2), ContentType: "image/png", Ext: ".png"},
		}
		id, err := s.restaurants.Create(s.ctx, s.accounts[sp.owner], sp.in, uploads)
		if err != nil {
			return fmt.Errorf("create %s: %w", sp.in.Name, err)
		}
		ids[sp.in.Name] = id
		fmt.Printf("restaurant %-24s id=%d\n", sp.in.Name, id)
	}
	klong, somtam, noodles := ids["ครัวริมคลอง"], ids["ส้มตำหน้าตลาด"], ids["ก๋วยเตี๋ยวเรือหน้าวัด"]
	mookrata, cafe := ids["Midnight Moo Kra Ta"], ids["Sabai 24h Café"]

	// Upcoming reservations. ครัวริมคลอง is nearly full at lunch on the next
	// open day, so "Limited Seats Left" shows up.
	lunchDay := nextDay(1, time.Monday)
	for _, r := range []struct {
		who        string
		restaurant int64
		pax        int
		from, to   time.Time
	}{
		{"Alice", klong, 4, at(lunchDay, "12:00"), at(lunchDay, "13:00")},
		{"Bob", klong, 4, at(lunchDay, "12:00"), at(lunchDay, "13:30")},
		{"Carol", klong, 2, at(lunchDay, "18:00"), at(lunchDay, "19:30")},
		{"Alice", mookrata, 6, at(nextDay(2), "23:00"), at(nextDay(3), "01:00")}, // past midnight
		{"Dan", somtam, 3, at(nextDay(1), "11:30"), at(nextDay(1), "12:30")},
		{"Bob", cafe, 2, at(nextDay(3), "03:00"), at(nextDay(3), "04:00")},
		{"Malee", klong, 2, at(nextDay(5, time.Monday), "19:00"), at(nextDay(5, time.Monday), "20:30")},
		{"Somchai", mookrata, 4, at(nextDay(4), "19:00"), at(nextDay(4), "21:00")},
	} {
		if _, err := s.reservations.Create(s.ctx, s.accounts[r.who], r.restaurant, reservationmodel.Input{
			Pax: r.pax, StartsAt: r.from, EndsAt: r.to,
		}); err != nil {
			return fmt.Errorf("reserve for %s: %w", r.who, err)
		}
	}

	// Past visits can't be booked through the API (R-BOOK-3), so insert them
	// directly. They make the matching reviews "verified" (R-REVIEW-4).
	for _, p := range []struct {
		who        string
		restaurant int64
		daysAgo    int
	}{
		{"Alice", klong, 6}, {"Bob", klong, 13}, {"Carol", somtam, 3}, {"Dan", noodles, 2}, {"Alice", noodles, 9},
	} {
		start := at(time.Now().In(bkk).AddDate(0, 0, -p.daysAgo), "12:00")
		if _, err := s.pool.Exec(s.ctx, `
			INSERT INTO reservations (restaurant_id, account_id, pax, starts_at, ends_at, created_at)
			VALUES ($1, $2, 2, $3, $4, $3::timestamptz - interval '3 days')`,
			p.restaurant, s.accounts[p.who], start, start.Add(time.Hour)); err != nil {
			return err
		}
	}

	for _, r := range []struct {
		who        string
		restaurant int64
		rating     int
		body       string
	}{
		{"Alice", klong, 5, "The green curry is the best I've had in Bangkok. Tiny place, book early."},
		{"Bob", klong, 5, "Sat by the canal, great service."},
		{"Carol", klong, 4, "Lovely food, a bit slow at peak time."},
		{"Carol", somtam, 5, "Proper spicy som tam. The grilled chicken is a must."},
		{"Dan", somtam, 4, "Good value, loud and fun."},
		{"Alice", somtam, 4, "Great papaya salad."},
		{"Bob", somtam, 5, "Came twice in one week."},
		{"Dan", noodles, 5, "Ate 12 bowls. No regrets."},
		{"Alice", noodles, 4, "Rich broth, quick service."},
		{"Carol", noodles, 3, "Good but the queue is long."},
		{"Bob", cafe, 5, "Only place open at 3 a.m. with decent coffee."},
		{"Dan", mookrata, 3, "Fun night out, the grill smoke gets everywhere."},
	} {
		if _, err := s.reviews.Upsert(s.ctx, s.accounts[r.who], r.restaurant, reviewmodel.Input{Rating: r.rating, Body: r.body}); err != nil {
			return fmt.Errorf("review by %s: %w", r.who, err)
		}
	}

	fmt.Printf("\nseeded %d accounts (password %q), %d restaurants\n", len(s.accounts), password, len(ids))
	return nil
}

func ptr(v int) *int { return &v }

// nextDay returns the Bangkok date `days` from today, skipping any closed weekdays.
func nextDay(days int, skip ...time.Weekday) time.Time {
	d := time.Now().In(bkk).AddDate(0, 0, days)
	for {
		ok := true
		for _, w := range skip {
			if d.Weekday() == w {
				ok = false
			}
		}
		if ok {
			return d
		}
		d = d.AddDate(0, 0, 1)
	}
}

// at is hh:mm Bangkok time on day's date.
func at(day time.Time, hhmm string) time.Time {
	var h, m int
	fmt.Sscanf(hhmm, "%d:%d", &h, &m)
	y, mo, d := day.In(bkk).Date()
	return time.Date(y, mo, d, h, m, 0, 0, bkk)
}

// art draws a simple placeholder "photo": a diagonal gradient with soft
// circles, tinted by hue. Swap for real photos any time via the UI.
func art(hue float64, seed int) []byte { return artSized(960, 640, hue, seed) }

func artSized(w, hgt int, hue float64, seed int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, hgt))
	cx := []float64{0.25, 0.7, 0.5}[seed%3] * float64(w)
	cy := []float64{0.4, 0.3, 0.7}[seed%3] * float64(hgt)
	r := float64(hgt) * 0.28
	for y := range hgt {
		for x := range w {
			t := (float64(x)/float64(w) + float64(y)/float64(hgt)) / 2
			l := 0.35 + 0.35*t
			if dist := math.Hypot(float64(x)-cx, float64(y)-cy); dist < r {
				l += 0.15 * (1 - dist/r)
			}
			img.Set(x, y, hsl(hue+40*t, 0.55, l))
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func hsl(h, s, l float64) color.RGBA {
	h = math.Mod(math.Mod(h, 360)+360, 360) / 360
	f := func(n float64) uint8 {
		k := math.Mod(n+h*12, 12)
		a := s * math.Min(l, 1-l)
		return uint8(255 * (l - a*math.Max(-1, math.Min(math.Min(k-3, 9-k), 1))))
	}
	return color.RGBA{f(0), f(8), f(4), 255}
}
