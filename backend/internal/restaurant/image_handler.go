package restaurant

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"

	"restaurants/internal/auth"
	"restaurants/internal/platform/apperr"
	"restaurants/internal/platform/web"
	"restaurants/internal/restaurant/model"
)

const (
	// Per file, before server-side shrinking (imageproc). The browser shrinks
	// photos before upload, so real uploads are usually far smaller.
	maxImageBytes   = 10 << 20
	maxRequestBytes = model.MaxImages*maxImageBytes + 1<<20
)

// The real content type is sniffed from the bytes; the client's header is not trusted.
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func (h *Handler) AddImages(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	form, err := parseMultipart(c)
	if err != nil {
		return err
	}
	uploads, err := readUploads(form.File["images"])
	if err != nil {
		return err
	}
	me := auth.MustAccount(c).ID
	if err := h.svc.AddImages(c.Request.Context(), me, id, uploads); err != nil {
		return err
	}
	return h.writeImages(c, me, id, http.StatusCreated)
}

func (h *Handler) DeleteImage(c *gin.Context) error {
	id, imageID, err := restaurantAndImage(c)
	if err != nil {
		return err
	}
	me := auth.MustAccount(c).ID
	if err := h.svc.DeleteImage(c.Request.Context(), me, id, imageID); err != nil {
		return err
	}
	return h.writeImages(c, me, id, http.StatusOK)
}

func (h *Handler) SetCover(c *gin.Context) error {
	id, imageID, err := restaurantAndImage(c)
	if err != nil {
		return err
	}
	me := auth.MustAccount(c).ID
	if err := h.svc.SetCover(c.Request.Context(), me, id, imageID); err != nil {
		return err
	}
	return h.writeImages(c, me, id, http.StatusOK)
}

func (h *Handler) writeImages(c *gin.Context, me, id int64, status int) error {
	d, err := h.svc.Get(c.Request.Context(), me, id)
	if err != nil {
		return err
	}
	c.JSON(status, gin.H{"images": d.Images})
	return nil
}

func restaurantAndImage(c *gin.Context) (int64, int64, error) {
	id, err := web.PathID(c, "id")
	if err != nil {
		return 0, 0, err
	}
	imageID, err := web.PathID(c, "imageID")
	return id, imageID, err
}

func parseMultipart(c *gin.Context) (*multipart.Form, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBytes)
	if err := c.Request.ParseMultipartForm(8 << 20); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return nil, model.ErrTooLarge
		}
		return nil, apperr.New(apperr.BadRequest, "bad_request", "expected multipart/form-data")
	}
	return c.Request.MultipartForm, nil
}

func readUploads(files []*multipart.FileHeader) ([]model.Upload, error) {
	uploads := make([]model.Upload, 0, len(files))
	for _, fh := range files {
		if fh.Size > maxImageBytes {
			return nil, apperr.InvalidInput("%s is larger than 10 MB", fh.Filename)
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
			return nil, apperr.InvalidInput("%s must be a JPEG, PNG or WebP image", fh.Filename)
		}
		uploads = append(uploads, model.Upload{Data: data, ContentType: ct, Ext: ext})
	}
	return uploads, nil
}
