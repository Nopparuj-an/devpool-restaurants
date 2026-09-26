// Package apitest runs the full API against a throwaway database for
// integration tests, with a small cookie-keeping JSON client.
package apitest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"restaurants/internal/server"
	"restaurants/internal/testdb"
)

type Env struct {
	t   *testing.T
	DB  *pgxpool.Pool
	URL string
}

// New starts the API on a fresh database. Skips if no test DB is configured.
func New(t *testing.T) *Env {
	t.Helper()
	db := testdb.New(t)
	srv := httptest.NewServer(server.New(db, server.Options{}))
	t.Cleanup(srv.Close)
	return &Env{t: t, DB: db, URL: srv.URL}
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
