// Package service holds the restaurant rules: validation, ownership, the
// cover-image rules, and what happens on delete. SQL and object storage sit
// behind the Repository and ImageStore ports.
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"restaurants/internal/platform/apperr"
	"restaurants/internal/platform/imageproc"
	"restaurants/internal/restaurant/model"
)

// Repository is the outbound port for restaurant data.
type Repository interface {
	// LockOwner locks the restaurant row for the rest of the transaction and
	// returns its owner. Locking also serializes owner edits with bookings (ADR-0003).
	LockOwner(ctx context.Context, id int64) (int64, error) // apperr.ErrNotFound if missing
	Insert(ctx context.Context, ownerID int64, v model.Valid) (int64, error)
	UpdateDetails(ctx context.Context, id int64, v model.Valid) error
	ReplaceHours(ctx context.Context, id int64, shifts []model.Shift) error
	Delete(ctx context.Context, id int64) error

	ImageKeys(ctx context.Context, id int64) ([]string, error)
	ImageStats(ctx context.Context, id int64) (model.ImageStats, error)
	InsertImage(ctx context.Context, id int64, key string, isCover bool, position int) error
	DeleteImage(ctx context.Context, id, imageID int64) (key string, wasCover bool, err error) // ErrNotFound if missing
	PromoteFirstImage(ctx context.Context, id int64) error
	SetCover(ctx context.Context, id, imageID int64) error // ErrNotFound if missing

	// List returns one page and the total number of matches.
	List(ctx context.Context, q model.ListQuery) ([]model.Summary, int, error)
	Get(ctx context.Context, id int64) (model.Detail, error) // ErrNotFound if missing
	UpcomingReservations(ctx context.Context, id int64, now time.Time) (int, error)
}

// ImageStore is the outbound port for image objects (Garage, ADR-0005).
type ImageStore interface {
	Put(ctx context.Context, key, contentType string, body []byte) error
	Delete(ctx context.Context, keys ...string) error
}

type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Service is the inbound port used by the HTTP handlers. `me` is the
// logged-in account. Every change checks that `by` owns the restaurant or is
// an admin (R-REST-2, R-ADMIN-6).
type Service interface {
	Create(ctx context.Context, me int64, in model.Input, uploads []model.Upload) (int64, error)
	Update(ctx context.Context, by model.Actor, id int64, in model.Input) error
	Delete(ctx context.Context, by model.Actor, id int64) error
	AddImages(ctx context.Context, by model.Actor, id int64, uploads []model.Upload) error
	DeleteImage(ctx context.Context, by model.Actor, id, imageID int64) error
	SetCover(ctx context.Context, by model.Actor, id, imageID int64) error
	// List returns one page (limit ≤ 100, default 50) and the total number of matches.
	List(ctx context.Context, q model.ListQuery) ([]model.Summary, int, error)
	// Get returns one restaurant. viewer is the logged-in account ID or 0;
	// hidden restaurants are only shown to their owner and admins.
	Get(ctx context.Context, viewer int64, admin bool, id int64) (model.Detail, error)
}

type service struct {
	repo         Repository
	images       ImageStore
	tx           TxRunner
	imageBaseURL string
	now          func() time.Time
}

func New(repo Repository, images ImageStore, tx TxRunner, imageBaseURL string) Service {
	return &service{repo: repo, images: images, tx: tx, imageBaseURL: strings.TrimSuffix(imageBaseURL, "/"), now: time.Now}
}

func (s *service) lockOwned(ctx context.Context, id int64, by model.Actor) error {
	owner, err := s.repo.LockOwner(ctx, id)
	if err != nil {
		return err
	}
	if owner != by.ID && !by.Admin {
		return model.ErrNotOwner
	}
	return nil
}

// Create inserts a restaurant with its hours and images; the first upload
// becomes the cover (R-REST-1).
func (s *service) Create(ctx context.Context, me int64, in model.Input, uploads []model.Upload) (int64, error) {
	v, err := validate(in, true)
	if err != nil {
		return 0, err
	}
	if len(uploads) == 0 {
		return 0, model.ErrImageRequired
	}
	if len(uploads) > model.MaxImages {
		return 0, model.ErrTooManyImages
	}
	var (
		id   int64
		keys []string
	)
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if id, err = s.repo.Insert(ctx, me, v); err != nil {
			return err
		}
		if err := s.repo.ReplaceHours(ctx, id, v.Shifts); err != nil {
			return err
		}
		keys, err = s.putImages(ctx, id, uploads)
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
func (s *service) Update(ctx context.Context, by model.Actor, id int64, in model.Input) error {
	v, err := validate(in, false)
	if err != nil {
		return err
	}
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.lockOwned(ctx, id, by); err != nil {
			return err
		}
		if err := s.repo.UpdateDetails(ctx, id, v); err != nil {
			return err
		}
		return s.repo.ReplaceHours(ctx, id, v.Shifts)
	})
}

// Delete removes the restaurant; reservations, reviews and image rows
// cascade in the database (R-REST-5). Image objects go after commit.
func (s *service) Delete(ctx context.Context, by model.Actor, id int64) error {
	var keys []string
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.lockOwned(ctx, id, by); err != nil {
			return err
		}
		var err error
		if keys, err = s.repo.ImageKeys(ctx, id); err != nil {
			return err
		}
		return s.repo.Delete(ctx, id)
	})
	if err != nil {
		return err
	}
	s.deleteObjects(keys)
	return nil
}

func (s *service) AddImages(ctx context.Context, by model.Actor, id int64, uploads []model.Upload) error {
	if len(uploads) == 0 {
		return model.ErrNoImages
	}
	var keys []string
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.lockOwned(ctx, id, by); err != nil {
			return err
		}
		var err error
		keys, err = s.putImages(ctx, id, uploads)
		return err
	})
	if err != nil {
		s.deleteObjects(keys)
	}
	return err
}

// putImages stores uploads and inserts their rows. The first image becomes
// the cover if there is none. It returns the keys already written so the
// caller can clean up if the transaction fails.
func (s *service) putImages(ctx context.Context, id int64, uploads []model.Upload) ([]string, error) {
	stats, err := s.repo.ImageStats(ctx, id)
	if err != nil {
		return nil, err
	}
	if stats.Count+len(uploads) > model.MaxImages {
		return nil, model.ErrTooManyImages
	}
	var keys []string
	for i, u := range uploads {
		// Oversized photos are shrunk to 1600 px JPEG before storage.
		img, err := imageproc.Normalize(u.Data)
		switch {
		case errors.Is(err, imageproc.ErrTooManyPx):
			return keys, model.ErrImagePixels
		case err != nil:
			return keys, model.ErrImageType
		}
		key := fmt.Sprintf("restaurants/%d/%s%s", id, randomHex(16), img.Ext)
		if err := s.images.Put(ctx, key, img.ContentType, img.Data); err != nil {
			return keys, err
		}
		keys = append(keys, key)
		if err := s.repo.InsertImage(ctx, id, key, !stats.HasCover && i == 0, stats.NextPosition+i); err != nil {
			return keys, err
		}
	}
	return keys, nil
}

// DeleteImage removes one image. The last one can't go (R-REST-1); removing
// the cover promotes the next image.
func (s *service) DeleteImage(ctx context.Context, by model.Actor, id, imageID int64) error {
	var key string
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.lockOwned(ctx, id, by); err != nil {
			return err
		}
		stats, err := s.repo.ImageStats(ctx, id)
		if err != nil {
			return err
		}
		var wasCover bool
		if key, wasCover, err = s.repo.DeleteImage(ctx, id, imageID); err != nil {
			return err
		}
		if stats.Count == 1 {
			return model.ErrLastImage // rolls back the delete
		}
		if wasCover {
			return s.repo.PromoteFirstImage(ctx, id)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.deleteObjects([]string{key})
	return nil
}

func (s *service) SetCover(ctx context.Context, by model.Actor, id, imageID int64) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.lockOwned(ctx, id, by); err != nil {
			return err
		}
		return s.repo.SetCover(ctx, id, imageID)
	})
}

func (s *service) List(ctx context.Context, q model.ListQuery) ([]model.Summary, int, error) {
	switch q.Sort {
	case "", "top_rated", "most_reviewed", "newest":
	default:
		return nil, 0, model.ErrInvalidSort
	}
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 50
	}
	q.Offset = max(q.Offset, 0)
	q.Q = strings.TrimSpace(q.Q)
	list, total, err := s.repo.List(ctx, q)
	for i := range list {
		list[i].CoverURL = s.url(list[i].CoverKey)
	}
	return list, total, err
}

func (s *service) Get(ctx context.Context, viewer int64, admin bool, id int64) (model.Detail, error) {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return d, err
	}
	owner := viewer != 0 && viewer == d.Owner.ID
	if d.Hidden() && !owner && !admin {
		return model.Detail{}, apperr.ErrNotFound
	}
	if !owner && !admin {
		d.BanReason = ""
	}
	d.CoverURL = s.url(d.CoverKey)
	for i := range d.Images {
		d.Images[i].URL = s.url(d.Images[i].Key)
	}
	d.IsOwner = owner
	d.CanManage = owner || admin
	if d.CanManage {
		n, err := s.repo.UpcomingReservations(ctx, id, s.now())
		if err != nil {
			return d, err
		}
		d.UpcomingReservations = &n
	}
	return d, nil
}

func (s *service) url(key string) string {
	if key == "" {
		return ""
	}
	return s.imageBaseURL + "/" + key
}

// deleteObjects removes objects outside any transaction. A failure only
// leaks storage, so it is logged, not returned.
func (s *service) deleteObjects(keys []string) {
	if len(keys) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.images.Delete(ctx, keys...); err != nil {
		slog.Warn("delete image objects", "keys", keys, "err", err)
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
