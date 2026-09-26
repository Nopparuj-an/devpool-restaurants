package restaurant

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"restaurants/internal/auth"
	"restaurants/internal/httpx"
)

const (
	maxImageBytes   = 5 << 20
	maxRequestBytes = maxImagesPerRestaurant*maxImageBytes + 1<<20
)

// Upload is a validated image file from a multipart request.
type Upload struct {
	Data        []byte
	ContentType string
	Ext         string
}

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/restaurants", httpx.Handler(h.list))
	mux.Handle("GET /api/restaurants/{id}", httpx.Handler(h.get))
	mux.Handle("POST /api/restaurants", auth.Require(h.create))
	mux.Handle("PUT /api/restaurants/{id}", auth.Require(h.update))
	mux.Handle("DELETE /api/restaurants/{id}", auth.Require(h.delete))
	mux.Handle("GET /api/me/restaurants", auth.Require(h.mine))
	mux.Handle("POST /api/restaurants/{id}/images", auth.Require(h.addImages))
	mux.Handle("DELETE /api/restaurants/{id}/images/{imageID}", auth.Require(h.deleteImage))
	mux.Handle("PUT /api/restaurants/{id}/images/{imageID}/cover", auth.Require(h.setCover))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, err := h.svc.List(r.Context(), ListQuery{
		Sort: q.Get("sort"), Q: q.Get("q"), Cuisine: q.Get("cuisine"), Limit: limit, Offset: offset,
	})
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"restaurants": list})
	return nil
}

func (h *Handler) mine(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	list, err := h.svc.List(r.Context(), ListQuery{Sort: "newest", OwnerID: me.ID, Limit: 100})
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"restaurants": list})
	return nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	var viewer int64
	if me, ok := auth.AccountFrom(r.Context()); ok {
		viewer = me.ID
	}
	d, err := h.svc.Get(r.Context(), viewer, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, d)
	return nil
}

// create takes multipart/form-data: a "data" field with the Input JSON and
// one or more "images" files (the first becomes the cover).
func (h *Handler) create(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	form, err := parseMultipart(w, r)
	if err != nil {
		return err
	}
	var in Input
	dec := json.NewDecoder(strings.NewReader(firstValue(form, "data")))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return httpx.NewError(http.StatusBadRequest, "bad_request", `"data" must be the restaurant JSON: `+err.Error())
	}
	uploads, err := readUploads(form.File["images"])
	if err != nil {
		return err
	}
	id, err := h.svc.Create(r.Context(), me.ID, in, uploads)
	if err != nil {
		return err
	}
	d, err := h.svc.Get(r.Context(), me.ID, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, d)
	return nil
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	var in Input
	if err := httpx.Decode(w, r, &in); err != nil {
		return err
	}
	if err := h.svc.Update(r.Context(), me.ID, id, in); err != nil {
		return err
	}
	d, err := h.svc.Get(r.Context(), me.ID, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, d)
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

func (h *Handler) addImages(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	form, err := parseMultipart(w, r)
	if err != nil {
		return err
	}
	uploads, err := readUploads(form.File["images"])
	if err != nil {
		return err
	}
	if err := h.svc.AddImages(r.Context(), me.ID, id, uploads); err != nil {
		return err
	}
	return h.writeImages(w, r, me.ID, id, http.StatusCreated)
}

func (h *Handler) deleteImage(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	id, imageID, err := restaurantAndImage(r)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteImage(r.Context(), me.ID, id, imageID); err != nil {
		return err
	}
	return h.writeImages(w, r, me.ID, id, http.StatusOK)
}

func (h *Handler) setCover(w http.ResponseWriter, r *http.Request, me auth.Account) error {
	id, imageID, err := restaurantAndImage(r)
	if err != nil {
		return err
	}
	if err := h.svc.SetCover(r.Context(), me.ID, id, imageID); err != nil {
		return err
	}
	return h.writeImages(w, r, me.ID, id, http.StatusOK)
}

func (h *Handler) writeImages(w http.ResponseWriter, r *http.Request, me, id int64, status int) error {
	d, err := h.svc.Get(r.Context(), me, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, status, map[string]any{"images": d.Images})
	return nil
}

func restaurantAndImage(r *http.Request) (int64, int64, error) {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return 0, 0, err
	}
	imageID, err := httpx.PathID(r, "imageID")
	return id, imageID, err
}

func parseMultipart(w http.ResponseWriter, r *http.Request) (*multipart.Form, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return nil, httpx.NewError(http.StatusRequestEntityTooLarge, "too_large", "upload too large")
		}
		return nil, httpx.NewError(http.StatusBadRequest, "bad_request", "expected multipart/form-data")
	}
	return r.MultipartForm, nil
}

func firstValue(form *multipart.Form, key string) string {
	if v := form.Value[key]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// readUploads checks size and sniffs the real content type (the client's
// Content-Type header is not trusted).
func readUploads(files []*multipart.FileHeader) ([]Upload, error) {
	uploads := make([]Upload, 0, len(files))
	for _, fh := range files {
		if fh.Size > maxImageBytes {
			return nil, httpx.Invalid("%s is larger than 5 MB", fh.Filename)
		}
		f, err := fh.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, err
		}
		ct := http.DetectContentType(data)
		ext, ok := allowedImageTypes[ct]
		if !ok {
			return nil, httpx.Invalid("%s must be a JPEG, PNG or WebP image", fh.Filename)
		}
		uploads = append(uploads, Upload{Data: data, ContentType: ct, Ext: ext})
	}
	return uploads, nil
}
