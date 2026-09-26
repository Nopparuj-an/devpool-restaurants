package model

import "restaurants/internal/platform/apperr"

var (
	ErrNotOwner      = apperr.New(apperr.Forbidden, "R-REST-2", "only the owner can change this restaurant")
	ErrImageRequired = apperr.New(apperr.Invalid, "R-REST-1", "at least one image is required")
	ErrLastImage     = apperr.New(apperr.Invalid, "R-REST-1", "a restaurant needs at least one image")
	ErrHoursRequired = apperr.New(apperr.Invalid, "R-REST-1", "opening hours are required")
	ErrCutoff        = apperr.New(apperr.Invalid, "R-REST-3", "cancel cutoff must be at least 30 minutes, in 15-minute steps, at most 7 days")
	ErrMaxDuration   = apperr.New(apperr.Invalid, "R-REST-6", "max reservation length must be 15 minutes to 24 hours, in 15-minute steps")
	ErrShiftPerDay   = apperr.New(apperr.Invalid, "R-HOURS-1", "hours need at most one shift per weekday (0 to 6)")
	ErrShiftGrid     = apperr.New(apperr.Invalid, "R-HOURS-2", "opening times must be HH:MM on the 15-minute grid")
	ErrTooManyImages = apperr.InvalidInput("at most %d images per restaurant", MaxImages)
	ErrNoImages      = apperr.InvalidInput("no images uploaded")
	ErrTooLarge      = apperr.New(apperr.TooLarge, "too_large", "upload too large")
	ErrInvalidSort   = apperr.InvalidInput("sort must be top_rated, most_reviewed or newest")
)
