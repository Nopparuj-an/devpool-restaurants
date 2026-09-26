// Package apitest runs the full API against a throwaway database for
// integration tests, with a small cookie-keeping JSON client.
package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gin-gonic/gin"

	"restaurants/internal/platform/storage"
	"restaurants/internal/server"
	"restaurants/internal/testutil/testdb"
)

type Env struct {
	t      *testing.T
	DB     *pgxpool.Pool
	URL    string
	Images *storage.Memory
}

// New starts the API on a fresh database. Skips if no test DB is configured.
func New(t *testing.T) *Env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testdb.New(t)
	images := storage.NewMemory()
	srv := httptest.NewServer(server.New(db, server.Options{Images: images, ImageBaseURL: "/images"}))
	t.Cleanup(srv.Close)
	return &Env{t: t, DB: db, URL: srv.URL, Images: images}
}

// Client is one browser: it keeps its own session cookie.
type Client struct {
	env  *Env
	http *http.Client
}

func (e *Env) Client() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{env: e, http: &http.Client{Jar: jar}}
}

// Signup creates an account and leaves the client logged in as it.
func (e *Env) Signup(email, name string) *Client {
	e.t.Helper()
	c := e.Client()
	c.Do("POST", "/api/auth/signup", map[string]string{
		"email": email, "password": "password123", "display_name": name,
	}).Expect(http.StatusCreated)
	return c
}

type Response struct {
	t      *testing.T
	Status int
	Body   []byte
}

// Do sends body as JSON (nil = no body).
func (c *Client) Do(method, path string, body any) *Response {
	c.env.t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.env.t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.env.URL+path, r)
	if err != nil {
		c.env.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.Send(req)
}

// Send sends a prepared request (e.g. multipart) with this client's cookies.
func (c *Client) Send(req *http.Request) *Response {
	c.env.t.Helper()
	if req.URL.Host == "" {
		u, _ := req.URL.Parse(c.env.URL + req.URL.String())
		req.URL = u
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.env.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return &Response{t: c.env.t, Status: res.StatusCode, Body: b}
}

// Expect fails the test unless the status matches.
func (r *Response) Expect(status int) *Response {
	r.t.Helper()
	if r.Status != status {
		r.t.Fatalf("status = %d, want %d; body: %s", r.Status, status, r.Body)
	}
	return r
}

// ExpectError fails unless status and error code match.
func (r *Response) ExpectError(status int, code string) {
	r.t.Helper()
	r.Expect(status)
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	json.Unmarshal(r.Body, &body)
	if body.Error.Code != code {
		r.t.Fatalf("error code = %q, want %q; body: %s", body.Error.Code, code, r.Body)
	}
}

// JSON decodes the body into v.
func (r *Response) JSON(v any) {
	r.t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		r.t.Fatalf("decode %s: %v", r.Body, err)
	}
}

// File is one multipart file part.
type File struct {
	Field, Name string
	Data        []byte
}

// Multipart sends form fields and files as multipart/form-data.
func (c *Client) Multipart(method, path string, fields map[string]string, files ...File) *Response {
	c.env.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	for _, f := range files {
		part, _ := mw.CreateFormFile(f.Field, f.Name)
		part.Write(f.Data)
	}
	mw.Close()
	req, err := http.NewRequest(method, path, &buf)
	if err != nil {
		c.env.t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return c.Send(req)
}

// PNG returns a tiny valid PNG image.
func PNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := range img.Pix {
		img.Pix[i] = 0xcc
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

// RestaurantInput is a valid restaurant open 10:00–23:00 every day in
// Asia/Bangkok; override fields as needed.
func RestaurantInput(name string, seats int) map[string]any {
	hours := make([]map[string]any, 7)
	for d := range hours {
		hours[d] = map[string]any{"weekday": d, "open": "10:00", "close": "23:00"}
	}
	return map[string]any{
		"name": name, "description": "Test restaurant", "cuisine": "Thai",
		"location": "Bangkok", "seats": seats, "timezone": "Asia/Bangkok", "hours": hours,
	}
}

// CreateRestaurant creates a restaurant with one image and returns its ID.
func (c *Client) CreateRestaurant(input map[string]any) int64 {
	c.env.t.Helper()
	data, _ := json.Marshal(input)
	var out struct {
		ID int64 `json:"id"`
	}
	c.Multipart("POST", "/api/restaurants", map[string]string{"data": string(data)},
		File{"images", "cover.png", PNG()}).Expect(http.StatusCreated).JSON(&out)
	return out.ID
}

// MakeAdmin grants admin rights the way operators do: directly in the database.
func (e *Env) MakeAdmin(email string) {
	e.t.Helper()
	if _, err := e.DB.Exec(context.Background(), `UPDATE accounts SET is_admin = true WHERE email = $1`, email); err != nil {
		e.t.Fatal(err)
	}
}

// Login returns a new client logged in with the seed test password.
func (e *Env) Login(email string) *Client {
	e.t.Helper()
	c := e.Client()
	c.Do("POST", "/api/auth/login", map[string]string{"email": email, "password": "password123"}).Expect(http.StatusOK)
	return c
}
