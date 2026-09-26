package restaurant_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"strings"
	"testing"

	"restaurants/internal/testutil/apitest"
)

type detail struct {
	ID                   int64  `json:"id"`
	Name                 string `json:"name"`
	CoverURL             string `json:"cover_url"`
	IsOwner              bool   `json:"is_owner"`
	UpcomingReservations *int   `json:"upcoming_reservations"`
	CancelCutoffMinutes  int    `json:"cancel_cutoff_minutes"`
	Hours                []struct {
		Weekday     int
		Open, Close string
	} `json:"hours"`
	Images []struct {
		ID      int64  `json:"id"`
		URL     string `json:"url"`
		IsCover bool   `json:"is_cover"`
	} `json:"images"`
}

func createRaw(c *apitest.Client, input map[string]any, files ...apitest.File) *apitest.Response {
	data, _ := json.Marshal(input)
	return c.Multipart("POST", "/api/restaurants", map[string]string{"data": string(data)}, files...)
}

func TestCreateValidation(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	img := apitest.File{Field: "images", Name: "a.png", Data: apitest.PNG()}

	createRaw(env.Client(), apitest.RestaurantInput("X", 10), img).ExpectError(http.StatusUnauthorized, "unauthorized")
	createRaw(owner, apitest.RestaurantInput("X", 10)).ExpectError(http.StatusUnprocessableEntity, "R-REST-1")
	createRaw(owner, apitest.RestaurantInput("X", 10), apitest.File{Field: "images", Name: "a.png", Data: []byte("not an image")}).
		ExpectError(http.StatusUnprocessableEntity, "invalid_input")

	bad := func(mutate func(map[string]any)) map[string]any {
		in := apitest.RestaurantInput("X", 10)
		mutate(in)
		return in
	}
	createRaw(owner, bad(func(in map[string]any) { in["seats"] = 0 }), img).ExpectError(http.StatusUnprocessableEntity, "invalid_input")
	createRaw(owner, bad(func(in map[string]any) { in["cancel_cutoff_minutes"] = 15 }), img).ExpectError(http.StatusUnprocessableEntity, "R-REST-3")
	createRaw(owner, bad(func(in map[string]any) { in["max_reservation_minutes"] = 20 }), img).ExpectError(http.StatusUnprocessableEntity, "R-REST-6")
	createRaw(owner, bad(func(in map[string]any) { in["timezone"] = "Mars/Olympus" }), img).ExpectError(http.StatusUnprocessableEntity, "invalid_input")
	createRaw(owner, bad(func(in map[string]any) { in["hours"] = []any{} }), img).ExpectError(http.StatusUnprocessableEntity, "R-REST-1")
	createRaw(owner, bad(func(in map[string]any) {
		in["hours"] = []any{map[string]any{"weekday": 1, "open": "10:10", "close": "22:00"}}
	}), img).ExpectError(http.StatusUnprocessableEntity, "R-HOURS-2")
	createRaw(owner, bad(func(in map[string]any) {
		in["hours"] = []any{
			map[string]any{"weekday": 1, "open": "10:00", "close": "14:00"},
			map[string]any{"weekday": 1, "open": "17:00", "close": "22:00"},
		}
	}), img).ExpectError(http.StatusUnprocessableEntity, "R-HOURS-1")

	if n := env.Images.Len(); n != 0 {
		t.Errorf("failed creates left %d objects in storage", n)
	}
}

func TestCreateAndGet(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")

	in := apitest.RestaurantInput("Night Noodles", 12)
	in["hours"] = []any{map[string]any{"weekday": 5, "open": "18:00", "close": "02:00"}}
	var d detail
	createRaw(owner, in,
		apitest.File{Field: "images", Name: "cover.png", Data: apitest.PNG()},
		apitest.File{Field: "images", Name: "extra.png", Data: apitest.PNG()},
	).Expect(http.StatusCreated).JSON(&d)

	if !d.IsOwner || d.UpcomingReservations == nil || *d.UpcomingReservations != 0 {
		t.Errorf("owner view: is_owner=%v upcoming=%v", d.IsOwner, d.UpcomingReservations)
	}
	if d.CancelCutoffMinutes != 30 {
		t.Errorf("default cutoff = %d, want 30", d.CancelCutoffMinutes)
	}
	if len(d.Hours) != 1 || d.Hours[0].Open != "18:00" || d.Hours[0].Close != "02:00" {
		t.Errorf("hours = %+v", d.Hours)
	}
	if len(d.Images) != 2 || !d.Images[0].IsCover || d.Images[0].URL != d.CoverURL ||
		!strings.HasPrefix(d.CoverURL, fmt.Sprintf("/images/restaurants/%d/", d.ID)) {
		t.Errorf("images = %+v cover = %q", d.Images, d.CoverURL)
	}
	if env.Images.Len() != 2 {
		t.Errorf("stored objects = %d, want 2", env.Images.Len())
	}

	// Public view hides owner-only fields.
	var pub detail
	env.Client().Do("GET", fmt.Sprintf("/api/restaurants/%d", d.ID), nil).Expect(http.StatusOK).JSON(&pub)
	if pub.IsOwner || pub.UpcomingReservations != nil {
		t.Errorf("public view leaked owner fields: %+v", pub)
	}
	env.Client().Do("GET", "/api/restaurants/999999", nil).ExpectError(http.StatusNotFound, "not_found")
	env.Client().Do("GET", "/api/restaurants/abc", nil).ExpectError(http.StatusNotFound, "not_found")
}

func TestOnlyOwnerCanModify(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	other := env.Signup("other@example.com", "Other")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Mine", 10))
	path := fmt.Sprintf("/api/restaurants/%d", id)

	other.Do("PUT", path, apitest.RestaurantInput("Stolen", 10)).ExpectError(http.StatusForbidden, "R-REST-2")
	other.Do("DELETE", path, nil).ExpectError(http.StatusForbidden, "R-REST-2")
	other.Multipart("POST", path+"/images", nil, apitest.File{Field: "images", Name: "x.png", Data: apitest.PNG()}).
		ExpectError(http.StatusForbidden, "R-REST-2")

	var d detail
	owner.Do("PUT", path, apitest.RestaurantInput("Renamed", 20)).Expect(http.StatusOK).JSON(&d)
	if d.Name != "Renamed" {
		t.Errorf("name = %q", d.Name)
	}
}

func TestImages(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Pics", 10))
	path := fmt.Sprintf("/api/restaurants/%d", id)

	var imgs struct {
		Images []struct {
			ID      int64 `json:"id"`
			IsCover bool  `json:"is_cover"`
		} `json:"images"`
	}
	owner.Multipart("POST", path+"/images", nil, apitest.File{Field: "images", Name: "b.png", Data: apitest.PNG()}).
		Expect(http.StatusCreated).JSON(&imgs)
	if len(imgs.Images) != 2 {
		t.Fatalf("images = %+v", imgs.Images)
	}
	first, second := imgs.Images[0].ID, imgs.Images[1].ID

	owner.Do("PUT", fmt.Sprintf("%s/images/%d/cover", path, second), nil).Expect(http.StatusOK).JSON(&imgs)
	if imgs.Images[0].ID != second || !imgs.Images[0].IsCover {
		t.Fatalf("after set cover: %+v", imgs.Images)
	}

	// Deleting the cover promotes the remaining image.
	owner.Do("DELETE", fmt.Sprintf("%s/images/%d", path, second), nil).Expect(http.StatusOK).JSON(&imgs)
	if len(imgs.Images) != 1 || imgs.Images[0].ID != first || !imgs.Images[0].IsCover {
		t.Fatalf("after delete cover: %+v", imgs.Images)
	}
	owner.Do("DELETE", fmt.Sprintf("%s/images/%d", path, first), nil).ExpectError(http.StatusUnprocessableEntity, "R-REST-1")
	if env.Images.Len() != 1 {
		t.Errorf("stored objects = %d, want 1", env.Images.Len())
	}
}

func TestDeleteCascades(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Doomed", 10))

	owner.Do("DELETE", fmt.Sprintf("/api/restaurants/%d", id), nil).Expect(http.StatusNoContent)
	owner.Do("GET", fmt.Sprintf("/api/restaurants/%d", id), nil).ExpectError(http.StatusNotFound, "not_found")
	if env.Images.Len() != 0 {
		t.Errorf("objects left after delete: %d", env.Images.Len())
	}
}

func TestListFilters(t *testing.T) {
	env := apitest.New(t)
	alice := env.Signup("alice@example.com", "Alice")
	bob := env.Signup("bob@example.com", "Bob")

	pad := apitest.RestaurantInput("Pad Thai 100%", 10)
	alice.CreateRestaurant(pad)
	sushi := apitest.RestaurantInput("Sushi Bar", 10)
	sushi["cuisine"] = "Japanese"
	bob.CreateRestaurant(sushi)

	names := func(c *apitest.Client, path string) []string {
		var out struct {
			Restaurants []struct{ Name string } `json:"restaurants"`
		}
		c.Do("GET", path, nil).Expect(http.StatusOK).JSON(&out)
		var n []string
		for _, r := range out.Restaurants {
			n = append(n, r.Name)
		}
		return n
	}
	check := func(got []string, want ...string) {
		t.Helper()
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	anon := env.Client()
	check(names(anon, "/api/restaurants?sort=newest"), "Sushi Bar", "Pad Thai 100%")
	check(names(anon, "/api/restaurants?q=sushi"), "Sushi Bar")
	check(names(anon, "/api/restaurants?q=100%25"), "Pad Thai 100%") // % is literal, not a wildcard
	check(names(anon, "/api/restaurants?q=0%25"), "Pad Thai 100%")
	check(names(anon, "/api/restaurants?q=%25%25%25"))
	check(names(anon, "/api/restaurants?cuisine=japanese"), "Sushi Bar")
	check(names(alice, "/api/me/restaurants"), "Pad Thai 100%")
	anon.Do("GET", "/api/restaurants?sort=bogus", nil).ExpectError(http.StatusUnprocessableEntity, "invalid_input")
}

func TestPaginationAndSearch(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	for i, cuisine := range []string{"Thai", "Japanese", "Thai"} {
		in := apitest.RestaurantInput(fmt.Sprintf("Place %d", i), 10)
		in["cuisine"] = cuisine
		owner.CreateRestaurant(in)
	}
	type page struct {
		Restaurants []struct{ Name string } `json:"restaurants"`
		Total       int                     `json:"total"`
	}
	get := func(q string) page {
		t.Helper()
		var p page
		env.Client().Do("GET", "/api/restaurants?"+q, nil).Expect(http.StatusOK).JSON(&p)
		return p
	}
	if p := get("sort=newest&limit=2"); p.Total != 3 || len(p.Restaurants) != 2 || p.Restaurants[0].Name != "Place 2" {
		t.Errorf("page 1 = %+v", p)
	}
	if p := get("sort=newest&limit=2&offset=2"); p.Total != 3 || len(p.Restaurants) != 1 || p.Restaurants[0].Name != "Place 0" {
		t.Errorf("page 2 = %+v", p)
	}
	if p := get("limit=2&offset=10"); p.Total != 3 || len(p.Restaurants) != 0 {
		t.Errorf("past the end = %+v", p)
	}
	// q searches cuisine as well as name.
	if p := get("q=japan"); p.Total != 1 || p.Restaurants[0].Name != "Place 1" {
		t.Errorf("q=japan = %+v", p)
	}
	if p := get("q=thai&limit=1"); p.Total != 2 || len(p.Restaurants) != 1 {
		t.Errorf("q=thai = %+v", p)
	}
}

func TestLargePhotoIsShrunk(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	var big bytes.Buffer
	png.Encode(&big, image.NewRGBA(image.Rect(0, 0, 3000, 2000)))

	var d detail
	createRaw(owner, apitest.RestaurantInput("Big", 10), apitest.File{Field: "images", Name: "big.png", Data: big.Bytes()}).
		Expect(http.StatusCreated).JSON(&d)
	if !strings.HasSuffix(d.CoverURL, ".jpg") {
		t.Fatalf("cover = %s, want a .jpg", d.CoverURL)
	}
	stored := env.Images.Objects[strings.TrimPrefix(d.CoverURL, "/images/")]
	cfg, format, err := image.DecodeConfig(bytes.NewReader(stored))
	if err != nil || format != "jpeg" || cfg.Width != 1600 || cfg.Height != 1067 {
		t.Fatalf("stored %s %dx%d err=%v", format, cfg.Width, cfg.Height, err)
	}
}
