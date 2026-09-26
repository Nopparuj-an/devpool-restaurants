package profile_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"restaurants/internal/testutil/apitest"
)

type profile struct {
	ID              int64  `json:"id"`
	DisplayName     string `json:"display_name"`
	RestaurantCount int    `json:"restaurant_count"`
	ReviewCount     int    `json:"review_count"`
	Banned          bool   `json:"banned"`
}

type reviews struct {
	Reviews []struct {
		Rating     int `json:"rating"`
		Restaurant struct {
			ID       int64  `json:"id"`
			Name     string `json:"name"`
			CoverURL string `json:"cover_url"`
		} `json:"restaurant"`
		Hidden bool `json:"hidden"`
	} `json:"reviews"`
	Total int `json:"total"`
}

func id(t *testing.T, c *apitest.Client) int64 {
	t.Helper()
	var m struct{ ID int64 }
	c.Do("GET", "/api/me", nil).Expect(http.StatusOK).JSON(&m)
	return m.ID
}

func TestPublicProfile(t *testing.T) {
	env := apitest.New(t)
	admin := env.Signup("admin@example.com", "Admin")
	env.MakeAdmin("admin@example.com")
	owner := env.Signup("owner@example.com", "Olivia")
	critic := env.Signup("critic@example.com", "Chris")
	criticID := id(t, critic)
	a := owner.CreateRestaurant(apitest.RestaurantInput("Alpha", 10))
	b := owner.CreateRestaurant(apitest.RestaurantInput("Beta", 10))
	for _, r := range []int64{a, b} {
		critic.Do("PUT", fmt.Sprintf("/api/restaurants/%d/reviews/me", r), map[string]any{"rating": 4, "body": "Good"}).Expect(http.StatusCreated)
	}
	anon := env.Client()

	var p profile
	res := anon.Do("GET", fmt.Sprintf("/api/users/%d", criticID), nil).Expect(http.StatusOK)
	res.JSON(&p)
	if p.DisplayName != "Chris" || p.ReviewCount != 2 || p.RestaurantCount != 0 {
		t.Fatalf("profile = %+v", p)
	}
	if strings.Contains(string(res.Body), "critic@example.com") {
		t.Fatal("profile must not show the email (R-PRIV-1)")
	}
	var rv reviews
	anon.Do("GET", fmt.Sprintf("/api/users/%d/reviews?limit=1", criticID), nil).Expect(http.StatusOK).JSON(&rv)
	if rv.Total != 2 || len(rv.Reviews) != 1 || rv.Reviews[0].Restaurant.Name != "Beta" || rv.Reviews[0].Restaurant.CoverURL == "" {
		t.Fatalf("reviews = %+v", rv)
	}

	// Restaurants on a profile come from the normal list, filtered by owner.
	var list struct{ Total int }
	anon.Do("GET", fmt.Sprintf("/api/restaurants?owner_id=%d", id(t, owner)), nil).Expect(http.StatusOK).JSON(&list)
	if list.Total != 2 {
		t.Fatalf("owner's restaurants = %d", list.Total)
	}

	// A banned restaurant drops out for the public, not for admins.
	admin.Do("POST", fmt.Sprintf("/api/admin/restaurants/%d/ban", a), nil).Expect(http.StatusNoContent)
	anon.Do("GET", fmt.Sprintf("/api/users/%d/reviews", criticID), nil).Expect(http.StatusOK).JSON(&rv)
	if rv.Total != 1 {
		t.Fatalf("public reviews after ban = %d", rv.Total)
	}
	anon.Do("GET", fmt.Sprintf("/api/restaurants?owner_id=%d", id(t, owner)), nil).Expect(http.StatusOK).JSON(&list)
	if list.Total != 1 {
		t.Fatalf("public restaurants after ban = %d", list.Total)
	}
	admin.Do("GET", fmt.Sprintf("/api/users/%d/reviews", criticID), nil).Expect(http.StatusOK).JSON(&rv)
	if rv.Total != 2 || !rv.Reviews[1].Hidden {
		t.Fatalf("admin reviews after ban = %+v", rv)
	}

	// A banned user's profile is gone, except for admins (R-PROFILE-2).
	admin.Do("POST", fmt.Sprintf("/api/admin/users/%d/ban", criticID), nil).Expect(http.StatusOK)
	anon.Do("GET", fmt.Sprintf("/api/users/%d", criticID), nil).Expect(http.StatusNotFound)
	anon.Do("GET", fmt.Sprintf("/api/users/%d/reviews", criticID), nil).Expect(http.StatusNotFound)
	admin.Do("GET", fmt.Sprintf("/api/users/%d", criticID), nil).Expect(http.StatusOK).JSON(&p)
	if !p.Banned {
		t.Fatal("admins should see the ban")
	}
	anon.Do("GET", "/api/users/999999", nil).Expect(http.StatusNotFound)
}
