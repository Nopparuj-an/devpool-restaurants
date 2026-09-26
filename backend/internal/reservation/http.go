package reservation

import (
	"net/http"
	"time"

	"restaurants/internal/auth"
	"restaurants/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/restaurants/{id}/availability", httpx.Handler(h.availability))
	mux.Handle("POST /api/restaurants/{id}/reservations", auth.Require(h.create))
	mux.Handle("GET /api/restaurants/{id}/reservations", auth.Require(h.listForOwner))
	mux.Handle("GET /api/me/reservations", auth.Require(h.listMine))
	mux.Handle("GET /api/reservations/{id}", auth.Require(h.get))
	mux.Handle("PUT /api/reservations/{id}", auth.Require(h.update))
	mux.Handle("POST /api/reservations/{id}/cancel", auth.Require(h.cancel))
}

func decodeInput(w http.ResponseWriter, r *http.Request) (Input, error) {
	var in Input
	if err := httpx.Decode(w, r, &in); err != nil {
		return in, err
	}
	if in.StartsAt.IsZero() || in.EndsAt.IsZero() {
		return in, httpx.Invalid("starts_at and ends_at are required (RFC 3339)")
	}
	return in, nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	restaurantID, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	in, err := decodeInput(w, r)
	if err != nil {
		return err
	}
	id, err := h.svc.Create(r.Context(), me.ID, restaurantID, in)
	if err != nil {
		return err
	}
	return h.write(w, r, me, id, http.StatusCreated)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	return h.write(w, r, me, id, http.StatusOK)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	in, err := decodeInput(w, r)
	if err != nil {
		return err
	}
	if err := h.svc.Update(r.Context(), me.ID, id, in); err != nil {
		return err
	}
	return h.write(w, r, me, id, http.StatusOK)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Cancel(r.Context(), me.ID, id); err != nil {
		return err
	}
	return h.write(w, r, me, id, http.StatusOK)
}

func (h *Handler) write(w http.ResponseWriter, r *http.Request, me auth.Account, id int64, status int) error {
	res, err := h.svc.Get(r.Context(), me.ID, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, status, res)
	return nil
}

func (h *Handler) listMine(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	list, err := h.svc.ListMine(r.Context(), me.ID)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"reservations": list})
	return nil
}

// listForOwner defaults to the next 7 days when from/to are omitted.
func (h *Handler) listForOwner(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	from, to, err := timeRange(r, 7*24*time.Hour)
	if err != nil {
		return err
	}
	list, err := h.svc.ListForOwner(r.Context(), me.ID, id, from, to)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"reservations": list})
	return nil
}

// availability defaults to the next 24 hours when from/to are omitted.
func (h *Handler) availability(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	from, to, err := timeRange(r, 24*time.Hour)
	if err != nil {
		return err
	}
	a, err := h.svc.Availability(r.Context(), id, from, to)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, a)
	return nil
}

// timeRange reads ?from=&to= (RFC 3339); from defaults to now, to to from+span.
func timeRange(r *http.Request, span time.Duration) (time.Time, time.Time, error) {
	parse := func(key string, fallback time.Time) (time.Time, error) {
		v := r.URL.Query().Get(key)
		if v == "" {
			return fallback, nil
		}
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return t, httpx.Invalid("%s must be an RFC 3339 timestamp", key)
		}
		return t, nil
	}
	from, err := parse("from", time.Now())
	if err != nil {
		return from, from, err
	}
	to, err := parse("to", from.Add(span))
	return from, to, err
}
