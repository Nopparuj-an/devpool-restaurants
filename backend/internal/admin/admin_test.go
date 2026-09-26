package admin_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"restaurants/internal/testutil/apitest"
)

type summary struct {
	ID          int64    `json:"id"`
	Rating      *float64 `json:"rating"`
	ReviewCount int      `json:"review_count"`
	Banned      bool     `json:"banned"`
	OwnerBanned bool     `json:"owner_banned"`
	BanReason   string   `json:"ban_reason"`
}

func rating(t *testing.T, c *apitest.Client, id int64) summary {
	t.Helper()
	var s summary
	c.Do("GET", fmt.Sprintf("/api/restaurants/%d", id), nil).Expect(http.StatusOK).JSON(&s)
	return s
}

func listed(t *testing.T, c *apitest.Client, id int64) bool {
	t.Helper()
	var out struct{ Restaurants []summary }
	c.Do("GET", "/api/restaurants?limit=100", nil).Expect(http.StatusOK).JSON(&out)
	for _, r := range out.Restaurants {
		if r.ID == id {
			return true
		}
	}
	return false
}

func review(c *apitest.Client, id int64, stars int) {
	c.Do("PUT", fmt.Sprintf("/api/restaurants/%d/reviews/me", id), map[string]any{"rating": stars, "body": "ok"}).Expect(http.StatusCreated)
}

func tomorrow(hh int) map[string]any {
	start := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour).Add(time.Duration(hh-7) * time.Hour) // hh:00 Bangkok
	return map[string]any{"pax": 2, "starts_at": start, "ends_at": start.Add(time.Hour)}
}

func TestAdminOnly(t *testing.T) {
	env := apitest.New(t)
	user := env.Signup("user@example.com", "User")
	user.Do("GET", "/api/admin/users", nil).ExpectError(http.StatusForbidden, "admin_only")
	env.Client().Do("GET", "/api/admin/users", nil).ExpectError(http.StatusUnauthorized, "unauthorized")

	var me struct {
		IsAdmin bool `json:"is_admin"`
	}
	user.Do("GET", "/api/me", nil).Expect(http.StatusOK).JSON(&me)
	if me.IsAdmin {
		t.Fatal("new accounts must not be admins")
	}
	env.MakeAdmin("user@example.com")
	user.Do("GET", "/api/me", nil).Expect(http.StatusOK).JSON(&me)
	if !me.IsAdmin {
		t.Fatal("is_admin not reported after grant")
	}
	user.Do("GET", "/api/admin/users", nil).Expect(http.StatusOK)
}

func TestUserListAndDetail(t *testing.T) {
	env := apitest.New(t)
	admin := env.Signup("admin@example.com", "Admin")
	env.MakeAdmin("admin@example.com")
	owner := env.Signup("owner@example.com", "Olivia Owner")
	env.Signup("bob@example.com", "Bob")
	env.Signup("carol@example.com", "Carol")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Olivia's", 10))

	var page struct {
		Users []struct {
			Email       string `json:"email"`
			Restaurants int    `json:"restaurants"`
		} `json:"users"`
		Total int `json:"total"`
	}
	admin.Do("GET", "/api/admin/users?limit=2", nil).Expect(http.StatusOK).JSON(&page)
	if page.Total != 4 || len(page.Users) != 2 || page.Users[0].Email != "carol@example.com" {
		t.Fatalf("page 1 = %+v", page)
	}
	admin.Do("GET", "/api/admin/users?q=olivia", nil).Expect(http.StatusOK).JSON(&page)
	if page.Total != 1 || page.Users[0].Restaurants != 1 {
		t.Fatalf("search by name = %+v", page)
	}
	admin.Do("GET", "/api/admin/users?status=nope", nil).ExpectError(http.StatusUnprocessableEntity, "invalid_input")

	var detail struct {
		ID               int64 `json:"id"`
		OwnedRestaurants []struct {
			ID int64 `json:"id"`
		} `json:"owned_restaurants"`
	}
	var ownerRow struct{ ID int64 }
	owner.Do("GET", "/api/me", nil).Expect(http.StatusOK).JSON(&ownerRow)
	admin.Do("GET", fmt.Sprintf("/api/admin/users/%d", ownerRow.ID), nil).Expect(http.StatusOK).JSON(&detail)
	if len(detail.OwnedRestaurants) != 1 || detail.OwnedRestaurants[0].ID != id {
		t.Fatalf("detail = %+v", detail)
	}
	admin.Do("GET", "/api/admin/users/999999", nil).ExpectError(http.StatusNotFound, "not_found")
}

// R-ADMIN-3: a banned user is locked out, their restaurants and reviews
// disappear, and ratings they affected are recomputed. Unbanning restores all.
func TestBanUser(t *testing.T) {
	env := apitest.New(t)
	admin := env.Signup("admin@example.com", "Admin")
	env.MakeAdmin("admin@example.com")
	alice := env.Signup("alice@example.com", "Alice")
	bob := env.Signup("bob@example.com", "Bob")
	carol := env.Signup("carol@example.com", "Carol")
	alicePlace := alice.CreateRestaurant(apitest.RestaurantInput("Alice's", 10))
	bobPlace := bob.CreateRestaurant(apitest.RestaurantInput("Bob's", 10))
	review(alice, bobPlace, 5)
	review(carol, bobPlace, 3)
	anon := env.Client()
	if s := rating(t, anon, bobPlace); *s.Rating != 4 || s.ReviewCount != 2 {
		t.Fatalf("before ban: %+v", s)
	}
	var aliceID struct{ ID int64 }
	alice.Do("GET", "/api/me", nil).Expect(http.StatusOK).JSON(&aliceID)

	admin.Do("POST", fmt.Sprintf("/api/admin/users/%d/ban", aliceID.ID), map[string]string{"reason": "spam"}).Expect(http.StatusOK)

	alice.Do("GET", "/api/me", nil).ExpectError(http.StatusUnauthorized, "unauthorized") // session ended
	env.Client().Do("POST", "/api/auth/login", map[string]string{"email": "alice@example.com", "password": "password123"}).
		ExpectError(http.StatusForbidden, "account_banned")
	anon.Do("GET", fmt.Sprintf("/api/restaurants/%d", alicePlace), nil).ExpectError(http.StatusNotFound, "not_found")
	if listed(t, anon, alicePlace) {
		t.Error("banned owner's restaurant still listed")
	}
	carol.Do("POST", fmt.Sprintf("/api/restaurants/%d/reservations", alicePlace), tomorrow(12)).ExpectError(http.StatusNotFound, "not_found")
	if s := rating(t, anon, bobPlace); *s.Rating != 3 || s.ReviewCount != 1 {
		t.Fatalf("after ban, Bob's rating should drop Alice's 5★: %+v", s)
	}
	var reviews struct{ Total int }
	anon.Do("GET", fmt.Sprintf("/api/restaurants/%d/reviews", bobPlace), nil).Expect(http.StatusOK).JSON(&reviews)
	if reviews.Total != 1 {
		t.Errorf("review total = %d, want 1", reviews.Total)
	}
	// Admins can still open the hidden restaurant.
	if s := rating(t, admin, alicePlace); !s.OwnerBanned {
		t.Errorf("admin view = %+v, want owner_banned", s)
	}

	admin.Do("POST", fmt.Sprintf("/api/admin/users/%d/unban", aliceID.ID), nil).Expect(http.StatusOK)
	env.Login("alice@example.com")
	if s := rating(t, anon, bobPlace); *s.Rating != 4 || s.ReviewCount != 2 {
		t.Fatalf("after unban: %+v", s)
	}
	if !listed(t, anon, alicePlace) {
		t.Error("restaurant not back after unban")
	}
}

func TestBanGuards(t *testing.T) {
	env := apitest.New(t)
	admin := env.Signup("admin@example.com", "Admin")
	env.MakeAdmin("admin@example.com")
	env.Signup("admin2@example.com", "Admin 2")
	env.MakeAdmin("admin2@example.com")
	var ids struct {
		Users []struct {
			ID    int64
			Email string
		}
	}
	admin.Do("GET", "/api/admin/users", nil).Expect(http.StatusOK).JSON(&ids)
	for _, u := range ids.Users {
		admin.Do("POST", fmt.Sprintf("/api/admin/users/%d/ban", u.ID), nil).ExpectError(http.StatusConflict, "R-ADMIN-5")
	}
	admin.Do("POST", "/api/admin/users/999999/ban", nil).ExpectError(http.StatusNotFound, "not_found")
}

// R-ADMIN-4: a banned restaurant is hidden and can't be booked or reviewed;
// existing bookings can still be cancelled; unbanning restores it.
func TestBanRestaurant(t *testing.T) {
	env := apitest.New(t)
	admin := env.Signup("admin@example.com", "Admin")
	env.MakeAdmin("admin@example.com")
	owner := env.Signup("owner@example.com", "Owner")
	carol := env.Signup("carol@example.com", "Carol")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Shady", 10))
	var booking struct{ ID int64 }
	carol.Do("POST", fmt.Sprintf("/api/restaurants/%d/reservations", id), tomorrow(12)).Expect(http.StatusCreated).JSON(&booking)

	admin.Do("POST", fmt.Sprintf("/api/admin/restaurants/%d/ban", id), map[string]string{"reason": "fake photos"}).Expect(http.StatusNoContent)

	anon := env.Client()
	anon.Do("GET", fmt.Sprintf("/api/restaurants/%d", id), nil).ExpectError(http.StatusNotFound, "not_found")
	anon.Do("GET", fmt.Sprintf("/api/restaurants/%d/availability", id), nil).ExpectError(http.StatusNotFound, "not_found")
	if listed(t, anon, id) {
		t.Error("banned restaurant still listed")
	}
	carol.Do("POST", fmt.Sprintf("/api/restaurants/%d/reservations", id), tomorrow(14)).ExpectError(http.StatusNotFound, "not_found")
	carol.Do("PUT", fmt.Sprintf("/api/restaurants/%d/reviews/me", id), map[string]any{"rating": 1, "body": "x"}).ExpectError(http.StatusNotFound, "not_found")
	if s := rating(t, owner, id); !s.Banned || s.BanReason != "fake photos" {
		t.Errorf("owner view = %+v, want banned with reason", s)
	}
	var mine struct{ Restaurants []summary }
	owner.Do("GET", "/api/me/restaurants", nil).Expect(http.StatusOK).JSON(&mine)
	if len(mine.Restaurants) != 1 || !mine.Restaurants[0].Banned {
		t.Errorf("my restaurants = %+v", mine)
	}
	carol.Do("POST", fmt.Sprintf("/api/reservations/%d/cancel", booking.ID), nil).Expect(http.StatusOK)

	var list struct {
		Restaurants []struct {
			BannedAt *time.Time `json:"banned_at"`
		}
		Total int
	}
	admin.Do("GET", "/api/admin/restaurants?status=banned", nil).Expect(http.StatusOK).JSON(&list)
	if list.Total != 1 || list.Restaurants[0].BannedAt == nil {
		t.Errorf("admin banned list = %+v", list)
	}

	admin.Do("POST", fmt.Sprintf("/api/admin/restaurants/%d/unban", id), nil).Expect(http.StatusNoContent)
	if s := rating(t, anon, id); s.Banned || s.BanReason != "" {
		t.Errorf("after unban = %+v", s)
	}
	admin.Do("POST", "/api/admin/restaurants/999999/ban", nil).ExpectError(http.StatusNotFound, "not_found")
}
