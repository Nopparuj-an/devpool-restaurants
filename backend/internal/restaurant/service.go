package restaurant

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"restaurants/internal/booking"
	"restaurants/internal/httpx"
	"restaurants/internal/storage"
)

const (
	maxImagesPerRestaurant = 10
	// bayesianPrior is m in ADR-0004: how many "average" reviews every
	// restaurant is assumed to start with when ranking by rating.
	bayesianPrior = 5
)

type Service struct {
	db           *pgxpool.Pool
	store        storage.Store
	imageBaseURL string
	now          func() time.Time
}

func NewService(db *pgxpool.Pool, store storage.Store, imageBaseURL string) *Service {
	return &Service{db: db, store: store, imageBaseURL: strings.TrimSuffix(imageBaseURL, "/"), now: time.Now}
}

func (s *Service) imageURL(key string) string {
	if key == "" {
		return ""
	}
	return s.imageBaseURL + "/" + key
}

// lockOwned locks the restaurant row for the rest of tx and checks that me
// owns it (R-REST-2). Locking also serializes owner edits with bookings (ADR-0003).
func lockOwned(ctx context.Context, tx pgx.Tx, id, me int64) error {
	var owner int64
	err := tx.QueryRow(ctx, `SELECT owner_id FROM restaurants WHERE id = $1 FOR UPDATE`, id).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	if owner != me {
		return httpx.NewError(403, "R-REST-2", "only the owner can change this restaurant")
	}
	return nil
}

// Create inserts a restaurant with its hours and images. The first upload
// becomes the cover (R-REST-1).
func (s *Service) Create(ctx context.Context, ownerID int64, in Input, uploads []Upload) (int64, error) {
	v, err := validate(in, true)
	if err != nil {
		return 0, err
	}
	if len(uploads) == 0 {
		return 0, httpx.NewError(422, "R-REST-1", "at least one image is required")
	}
	if len(uploads) > maxImagesPerRestaurant {
		return 0, httpx.Invalid("at most %d images per restaurant", maxImagesPerRestaurant)
	}

	var (
		id   int64
		keys []string
	)
	err = pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO restaurants (owner_id, name, description, cuisine, location, seats,
				cancel_cutoff_minutes, max_reservation_minutes, timezone)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
			ownerID, v.Name, v.Description, v.Cuisine, v.Location, v.Seats,
			v.cutoff, v.maxDuration, v.Timezone).Scan(&id)
		if err != nil {
			return err
		}
		if err := insertHours(ctx, tx, id, v.shifts); err != nil {
			return err
		}
		keys, err = s.putImages(ctx, tx, id, uploads)
		return err
	})
	if err != nil {
		s.deleteObjects(keys)
		return 0, err
	}
	return id, nil
}

// Update replaces the editable fields and hours. Existing reservations are
// never touched (R-REST-4). The timezone can't change after creation.
func (s *Service) Update(ctx context.Context, me, id int64, in Input) error {
	v, err := validate(in, false)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if err := lockOwned(ctx, tx, id, me); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE restaurants SET name = $2, description = $3, cuisine = $4, location = $5,
				seats = $6, cancel_cutoff_minutes = $7, max_reservation_minutes = $8, updated_at = now()
			WHERE id = $1`,
			id, v.Name, v.Description, v.Cuisine, v.Location, v.Seats, v.cutoff, v.maxDuration)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM restaurant_hours WHERE restaurant_id = $1`, id); err != nil {
			return err
		}
		return insertHours(ctx, tx, id, v.shifts)
	})
}

// Delete removes the restaurant; reservations, reviews and image rows
// cascade (R-REST-5). Image objects are deleted after commit, best effort.
func (s *Service) Delete(ctx context.Context, me, id int64) error {
	var keys []string
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if err := lockOwned(ctx, tx, id, me); err != nil {
			return err
		}
		rows, _ := tx.Query(ctx, `SELECT object_key FROM restaurant_images WHERE restaurant_id = $1`, id)
		var err error
		if keys, err = pgx.CollectRows(rows, pgx.RowTo[string]); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `DELETE FROM restaurants WHERE id = $1`, id)
		return err
	})
	if err != nil {
		return err
	}
	s.deleteObjects(keys)
	return nil
}

func insertHours(ctx context.Context, tx pgx.Tx, id int64, shifts []shift) error {
	for _, sh := range shifts {
		_, err := tx.Exec(ctx, `
			INSERT INTO restaurant_hours (restaurant_id, weekday, open_time, close_time)
			VALUES ($1, $2, $3::time, $4::time)`, id, sh.weekday, sh.open.String(), sh.close.String())
		if err != nil {
			return err
		}
	}
	return nil
}

// AddImages uploads more images to an owned restaurant.
func (s *Service) AddImages(ctx context.Context, me, id int64, uploads []Upload) error {
	if len(uploads) == 0 {
		return httpx.Invalid("no images uploaded")
	}
	var keys []string
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if err := lockOwned(ctx, tx, id, me); err != nil {
			return err
		}
		var err error
		keys, err = s.putImages(ctx, tx, id, uploads)
		return err
	})
	if err != nil {
		s.deleteObjects(keys)
	}
	return err
}

// putImages stores uploads in object storage and inserts their rows. The
// first image becomes the cover if the restaurant has none. It returns the
// keys already written so the caller can clean up if the transaction fails.
func (s *Service) putImages(ctx context.Context, tx pgx.Tx, id int64, uploads []Upload) ([]string, error) {
	var (
		count, nextPos int
		hasCover       bool
	)
	err := tx.QueryRow(ctx, `
		SELECT count(*), coalesce(max(position) + 1, 0), coalesce(bool_or(is_cover), false)
		FROM restaurant_images WHERE restaurant_id = $1`, id).Scan(&count, &nextPos, &hasCover)
	if err != nil {
		return nil, err
	}
	if count+len(uploads) > maxImagesPerRestaurant {
		return nil, httpx.Invalid("at most %d images per restaurant", maxImagesPerRestaurant)
	}

	var keys []string
	for i, u := range uploads {
		key := fmt.Sprintf("restaurants/%d/%s%s", id, randomHex(16), u.Ext)
		if err := s.store.Put(ctx, key, u.ContentType, u.Data); err != nil {
			return keys, err
		}
		keys = append(keys, key)
		isCover := !hasCover && i == 0
		_, err := tx.Exec(ctx, `
			INSERT INTO restaurant_images (restaurant_id, object_key, is_cover, position)
			VALUES ($1, $2, $3, $4)`, id, key, isCover, nextPos+i)
		if err != nil {
			return keys, err
		}
	}
	return keys, nil
}

// DeleteImage removes one image. The last image can't be removed (R-REST-1);
// removing the cover promotes the next image.
func (s *Service) DeleteImage(ctx context.Context, me, id, imageID int64) error {
	var key string
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if err := lockOwned(ctx, tx, id, me); err != nil {
			return err
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM restaurant_images WHERE restaurant_id = $1`, id).Scan(&count); err != nil {
			return err
		}
		var wasCover bool
		err := tx.QueryRow(ctx, `
			DELETE FROM restaurant_images WHERE id = $1 AND restaurant_id = $2
			RETURNING object_key, is_cover`, imageID, id).Scan(&key, &wasCover)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if count == 1 {
			return httpx.NewError(422, "R-REST-1", "a restaurant needs at least one image")
		}
		if wasCover {
			_, err = tx.Exec(ctx, `
				UPDATE restaurant_images SET is_cover = true
				WHERE id = (SELECT id FROM restaurant_images WHERE restaurant_id = $1 ORDER BY position LIMIT 1)`, id)
		}
		return err
	})
	if err != nil {
		return err
	}
	s.deleteObjects([]string{key})
	return nil
}

// SetCover makes imageID the restaurant's cover.
func (s *Service) SetCover(ctx context.Context, me, id, imageID int64) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if err := lockOwned(ctx, tx, id, me); err != nil {
			return err
		}
		var exists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM restaurant_images WHERE id = $1 AND restaurant_id = $2)`,
			imageID, id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return httpx.ErrNotFound
		}
		// Two statements: the one-cover unique index is checked row by row.
		if _, err := tx.Exec(ctx, `UPDATE restaurant_images SET is_cover = false WHERE restaurant_id = $1 AND is_cover`, id); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE restaurant_images SET is_cover = true WHERE id = $1`, imageID)
		return err
	})
}

// deleteObjects removes objects outside any transaction. Failures only leak
// storage, so they are logged, not returned.
func (s *Service) deleteObjects(keys []string) {
	if len(keys) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.store.Delete(ctx, keys...); err != nil {
		slog.Warn("delete image objects", "keys", keys, "err", err)
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// LockForBooking locks the restaurant row for the rest of tx (ADR-0003) and
// returns what the booking rules need, plus the owner's account ID.
func LockForBooking(ctx context.Context, tx pgx.Tx, id int64) (booking.Restaurant, int64, error) {
	var (
		r                   booking.Restaurant
		owner               int64
		cutoff, maxDuration int
		tz                  string
	)
	err := tx.QueryRow(ctx, `
		SELECT owner_id, seats, cancel_cutoff_minutes, max_reservation_minutes, timezone
		FROM restaurants WHERE id = $1 FOR UPDATE`, id).
		Scan(&owner, &r.Seats, &cutoff, &maxDuration, &tz)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, 0, httpx.ErrNotFound
	}
	if err != nil {
		return r, 0, err
	}
	r.CancelCutoff = time.Duration(cutoff) * time.Minute
	r.MaxDuration = time.Duration(maxDuration) * time.Minute
	if r.Location, err = time.LoadLocation(tz); err != nil {
		return r, 0, fmt.Errorf("restaurant %d timezone %q: %w", id, tz, err)
	}
	hours, err := loadHours(ctx, tx, id)
	if err != nil {
		return r, 0, err
	}
	for _, h := range hours {
		r.Hours[h.weekday] = &booking.Shift{Open: h.open, Close: h.close}
	}
	return r, owner, nil
}

func loadHours(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, id int64) ([]shift, error) {
	rows, _ := q.Query(ctx, `
		SELECT weekday, (extract(epoch FROM open_time) / 60)::int, (extract(epoch FROM close_time) / 60)::int
		FROM restaurant_hours WHERE restaurant_id = $1 ORDER BY weekday`, id)
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (shift, error) {
		var sh shift
		err := row.Scan(&sh.weekday, &sh.open, &sh.close)
		return sh, err
	})
}
