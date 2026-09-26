// Package service holds the profile rules: who can see a profile, and which
// of its restaurants and reviews count as public (R-PROFILE-*).
package service

import (
	"context"
	"strings"
	"time"

	"restaurants/internal/platform/apperr"
	"restaurants/internal/profile/model"
)

// Repository is the outbound port for profile data. includeHidden also
// counts and lists content customers can't see (R-ADMIN-3, -4).
type Repository interface {
	Profile(ctx context.Context, id int64, includeHidden bool) (model.Profile, error) // apperr.ErrNotFound if missing
	Reviews(ctx context.Context, id int64, includeHidden bool, now time.Time, limit, offset int) ([]model.Review, int, error)
}

// Service is the inbound port used by the HTTP handlers. viewer is the
// logged-in account ID or 0; admin says whether they are an admin.
type Service interface {
	Get(ctx context.Context, viewer int64, admin bool, id int64) (model.Profile, error)
	// Reviews returns one page (limit ≤ 100, default 20), newest first, and the total.
	Reviews(ctx context.Context, viewer int64, admin bool, id int64, limit, offset int) ([]model.Review, int, error)
}

type service struct {
	repo         Repository
	imageBaseURL string
	now          func() time.Time
}

func New(repo Repository, imageBaseURL string) Service {
	return &service{repo: repo, imageBaseURL: strings.TrimSuffix(imageBaseURL, "/"), now: time.Now}
}

// Hidden content is shown to the profile's owner and to admins only.
func includeHidden(viewer int64, admin bool, id int64) bool {
	return admin || (viewer != 0 && viewer == id)
}

func (s *service) Get(ctx context.Context, viewer int64, admin bool, id int64) (model.Profile, error) {
	p, err := s.repo.Profile(ctx, id, includeHidden(viewer, admin, id))
	if err != nil {
		return p, err
	}
	// A banned user's profile is gone for everyone but admins (R-PROFILE-2).
	if p.Banned && !admin {
		return model.Profile{}, apperr.ErrNotFound
	}
	return p, nil
}

func (s *service) Reviews(ctx context.Context, viewer int64, admin bool, id int64, limit, offset int) ([]model.Review, int, error) {
	if _, err := s.Get(ctx, viewer, admin, id); err != nil {
		return nil, 0, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	list, total, err := s.repo.Reviews(ctx, id, includeHidden(viewer, admin, id), s.now(), limit, max(offset, 0))
	for i := range list {
		if k := list[i].Restaurant.CoverKey; k != "" {
			list[i].Restaurant.CoverURL = s.imageBaseURL + "/" + k
		}
	}
	return list, total, err
}
