// Package review implements ratings and reviews (R-REVIEW-*).
//
// The restaurant's rating_sum and review_count are updated in the same
// transaction as the review itself (ADR-0004), so list pages can sort by
// rating without aggregating every review on each request.
package review

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"restaurants/internal/auth"
	"restaurants/internal/httpx"
)

type Author struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	// Email is only shown to the restaurant's owner (R-PRIV-2).
	Email string `json:"email,omitempty"`
}

type Review struct {
	ID        int64     `json:"id"`
	Rating    int       `json:"rating"`
	Body      string    `json:"body"`
	Verified  bool      `json:"verified"` // author has a completed reservation here (R-REVIEW-4)
	Author    Author    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Input struct {
	Rating int    `json:"rating"`
	Body   string `json:"body"`
}

type Service struct {
	db  *pgxpool.Pool
	now func() time.Time
}

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db, now: time.Now} }

// Upsert creates or replaces my review of a restaurant (R-REVIEW-2) and
// reports whether it was created.
func (s *Service) Upsert(ctx context.Context, me, restaurantID int64, in Input) (bool, error) {
	body := strings.TrimSpace(in.Body)
	if in.Rating < 1 || in.Rating > 5 {
		return false, httpx.NewError(422, "R-REVIEW-1", "rating must be a whole number from 1 to 5")
	}
	if n := utf8.RuneCountInString(body); n < 1 || n > 2000 {
		return false, httpx.NewError(422, "R-REVIEW-1", "review text must be 1–2000 characters")
	}

	var created bool
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if err := lockReviewable(ctx, tx, restaurantID, me); err != nil {
			return err
		}
		var old int
		err := tx.QueryRow(ctx, `
			SELECT rating FROM reviews WHERE restaurant_id = $1 AND account_id = $2`,
			restaurantID, me).Scan(&old)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			created = true
			if _, err := tx.Exec(ctx, `
				INSERT INTO reviews (restaurant_id, account_id, rating, body) VALUES ($1, $2, $3, $4)`,
				restaurantID, me, in.Rating, body); err != nil {
				return err
			}
			return adjustAggregate(ctx, tx, restaurantID, in.Rating, 1)
		case err != nil:
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE reviews SET rating = $3, body = $4, updated_at = now()
			WHERE restaurant_id = $1 AND account_id = $2`,
			restaurantID, me, in.Rating, body); err != nil {
			return err
		}
		return adjustAggregate(ctx, tx, restaurantID, in.Rating-old, 0)
	})
	return created, err
}

// Delete removes my review (R-REVIEW-7).
func (s *Service) Delete(ctx context.Context, me, restaurantID int64) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if err := lockRestaurant(ctx, tx, restaurantID); err != nil {
			return err
		}
		var old int
		err := tx.QueryRow(ctx, `
			DELETE FROM reviews WHERE restaurant_id = $1 AND account_id = $2 RETURNING rating`,
			restaurantID, me).Scan(&old)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		return adjustAggregate(ctx, tx, restaurantID, -old, -1)
	})
}

// lockRestaurant locks the restaurant row so aggregate updates for one
// restaurant are serialized, and returns its owner.
func lockRestaurant(ctx context.Context, tx pgx.Tx, id int64) error {
	_, err := lockOwner(ctx, tx, id)
	return err
}

func lockOwner(ctx context.Context, tx pgx.Tx, id int64) (int64, error) {
	var owner int64
	err := tx.QueryRow(ctx, `SELECT owner_id FROM restaurants WHERE id = $1 FOR UPDATE`, id).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.ErrNotFound
	}
	return owner, err
}

func lockReviewable(ctx context.Context, tx pgx.Tx, id, me int64) error {
	owner, err := lockOwner(ctx, tx, id)
	if err != nil {
		return err
	}
	if owner == me {
		return httpx.NewError(403, "R-REVIEW-3", "you can't review your own restaurant")
	}
	return nil
}

func adjustAggregate(ctx context.Context, tx pgx.Tx, id int64, sumDelta, countDelta int) error {
	_, err := tx.Exec(ctx, `
		UPDATE restaurants SET rating_sum = rating_sum + $2, review_count = review_count + $3
		WHERE id = $1`, id, sumDelta, countDelta)
	return err
}

const listSelect = `
	SELECT rv.id, rv.rating, rv.body, rv.created_at, rv.updated_at,
		a.id, a.display_name, a.email, r.owner_id,
		EXISTS (
			SELECT 1 FROM reservations v
			WHERE v.restaurant_id = rv.restaurant_id AND v.account_id = rv.account_id
				AND v.status = 'active' AND v.ends_at <= $2
		)
	FROM reviews rv
	JOIN accounts a ON a.id = rv.account_id
	JOIN restaurants r ON r.id = rv.restaurant_id`

func (s *Service) collect(rows pgx.Rows, viewer int64) ([]Review, error) {
	out := []Review{}
	defer rows.Close()
	for rows.Next() {
		var (
			rv    Review
			owner int64
		)
		err := rows.Scan(&rv.ID, &rv.Rating, &rv.Body, &rv.CreatedAt, &rv.UpdatedAt,
			&rv.Author.ID, &rv.Author.DisplayName, &rv.Author.Email, &owner, &rv.Verified)
		if err != nil {
			return nil, err
		}
		if viewer == 0 || viewer != owner {
			rv.Author.Email = "" // R-PRIV-1
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

// List returns a restaurant's reviews, most recently updated first.
func (s *Service) List(ctx context.Context, viewer, restaurantID int64, limit, offset int) ([]Review, error) {
	var exists bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM restaurants WHERE id = $1)`, restaurantID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, httpx.ErrNotFound
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.Query(ctx, listSelect+`
		WHERE rv.restaurant_id = $1
		ORDER BY rv.updated_at DESC, rv.id DESC
		LIMIT $3 OFFSET $4`, restaurantID, s.now(), limit, max(offset, 0))
	if err != nil {
		return nil, err
	}
	return s.collect(rows, viewer)
}

// Mine returns my review of a restaurant, or 404.
func (s *Service) Mine(ctx context.Context, me, restaurantID int64) (Review, error) {
	rows, err := s.db.Query(ctx, listSelect+`
		WHERE rv.restaurant_id = $1 AND rv.account_id = $3`, restaurantID, s.now(), me)
	if err != nil {
		return Review{}, err
	}
	list, err := s.collect(rows, me)
	if err != nil {
		return Review{}, err
	}
	if len(list) == 0 {
		return Review{}, httpx.ErrNotFound
	}
	return list[0], nil
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/restaurants/{id}/reviews", httpx.Handler(h.list))
	mux.Handle("GET /api/restaurants/{id}/reviews/me", auth.Require(h.mine))
	mux.Handle("PUT /api/restaurants/{id}/reviews/me", auth.Require(h.upsert))
	mux.Handle("DELETE /api/restaurants/{id}/reviews/me", auth.Require(h.delete))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	var viewer int64
	if me, ok := auth.AccountFrom(r.Context()); ok {
		viewer = me.ID
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	list, err := h.svc.List(r.Context(), viewer, id, limit, offset)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"reviews": list})
	return nil
}

func (h *Handler) mine(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	rv, err := h.svc.Mine(r.Context(), me.ID, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, rv)
	return nil
}

func (h *Handler) upsert(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	var in Input
	if err := httpx.Decode(w, r, &in); err != nil {
		return err
	}
	created, err := h.svc.Upsert(r.Context(), me.ID, id, in)
	if err != nil {
		return err
	}
	rv, err := h.svc.Mine(r.Context(), me.ID, id)
	if err != nil {
		return err
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.JSON(w, status, rv)
	return nil
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(r.Context(), me.ID, id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
