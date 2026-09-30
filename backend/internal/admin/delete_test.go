package admin_test

import (
	"fmt"
	"net/http"
	"testing"

	"restaurants/internal/testutil/apitest"
)

func meID(t *testing.T, c *apitest.Client) int64 {
	t.Helper()
	var me struct{ ID int64 }
	c.Do("GET", "/api/me", nil).Expect(http.StatusOK).JSON(&me)
	return me.ID
}

func deleted(t *testing.T, r *apitest.Response) int {
	t.Helper()
	var out struct{ Deleted int }
	r.Expect(http.StatusOK).JSON(&out)
	return out.Deleted
}

// R-ADMIN-8: deleting a user removes them and everything that cascades:
// their restaurants (with bookings, reviews and photos) and their own
// reviews, whose ratings are recomputed.
func TestDeleteUsers(t *testing.T) {
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
	review(carol, alicePlace, 4)
	carol.Do("POST", fmt.Sprintf("/api/restaurants/%d/reservations", alicePlace), tomorrow(12)).Expect(http.StatusCreated)
	photos := env.Images.Len()
	aliceID, carolID := meID(t, alice), meID(t, carol)

	// Missing IDs are skipped; duplicates count once.
	n := deleted(t, admin.Do("POST", "/api/admin/users/delete", map[string]any{"ids": []int64{aliceID, aliceID, 999999}}))
	if n != 1 {
		t.Fatalf("deleted = %d, want 1", n)
	}
	alice.Do("GET", "/api/me", nil).ExpectError(http.StatusUnauthorized, "session_expired")
	env.Client().Do("GET", fmt.Sprintf("/api/restaurants/%d", alicePlace), nil).ExpectError(http.StatusNotFound, "not_found")
	if s := rating(t, carol, bobPlace); *s.Rating != 3 || s.ReviewCount != 1 {
		t.Fatalf("Bob's rating should drop Alice's 5★: %+v", s)
	}
	if env.Images.Len() != photos-1 {
		t.Errorf("photos = %d, want %d (Alice's restaurant photo removed)", env.Images.Len(), photos-1)
	}
	var mine struct{ Total int }
	carol.Do("GET", "/api/me/reservations", nil).Expect(http.StatusOK).JSON(&mine)
	if mine.Total != 0 {
		t.Errorf("Carol's booking at Alice's restaurant should be gone, total = %d", mine.Total)
	}

	// Deleting again is harmless.
	if n := deleted(t, admin.Do("POST", "/api/admin/users/delete", map[string]any{"ids": []int64{aliceID}})); n != 0 {
		t.Errorf("second delete = %d, want 0", n)
	}
	if n := deleted(t, admin.Do("POST", "/api/admin/users/delete", map[string]any{"ids": []int64{carolID}})); n != 1 {
		t.Errorf("deleted = %d, want 1", n)
	}
}

func TestDeleteUserGuards(t *testing.T) {
	env := apitest.New(t)
	admin := env.Signup("admin@example.com", "Admin")
	env.MakeAdmin("admin@example.com")
	other := env.Signup("other@example.com", "Other admin")
	env.MakeAdmin("other@example.com")
	dan := env.Signup("dan@example.com", "Dan")
	adminID, otherID, danID := meID(t, admin), meID(t, other), meID(t, dan)

	admin.Do("POST", "/api/admin/users/delete", map[string]any{"ids": []int64{danID, adminID}}).ExpectError(http.StatusConflict, "R-ADMIN-8")
	admin.Do("POST", "/api/admin/users/delete", map[string]any{"ids": []int64{danID, otherID}}).ExpectError(http.StatusConflict, "R-ADMIN-8")
	dan.Do("GET", "/api/me", nil).Expect(http.StatusOK) // all or nothing: Dan is still there

	admin.Do("POST", "/api/admin/users/delete", map[string]any{"ids": []int64{}}).ExpectError(http.StatusUnprocessableEntity, "invalid_input")
	admin.Do("POST", "/api/admin/users/delete", map[string]any{"ids": []int64{-1}}).ExpectError(http.StatusUnprocessableEntity, "invalid_input")
	many := make([]int64, 501)
	for i := range many {
		many[i] = int64(i + 1000)
	}
	admin.Do("POST", "/api/admin/users/delete", map[string]any{"ids": many}).ExpectError(http.StatusUnprocessableEntity, "invalid_input")
	dan.Do("POST", "/api/admin/users/delete", map[string]any{"ids": []int64{adminID}}).ExpectError(http.StatusForbidden, "admin_only")
}

func TestDeleteRestaurants(t *testing.T) {
	env := apitest.New(t)
	admin := env.Signup("admin@example.com", "Admin")
	env.MakeAdmin("admin@example.com")
	owner := env.Signup("owner@example.com", "Owner")
	var ids []int64
	for i := range 3 {
		ids = append(ids, owner.CreateRestaurant(apitest.RestaurantInput(fmt.Sprintf("Place %d", i), 10)))
	}
	photos := env.Images.Len()

	n := deleted(t, admin.Do("POST", "/api/admin/restaurants/delete", map[string]any{"ids": ids[:2]}))
	if n != 2 {
		t.Fatalf("deleted = %d, want 2", n)
	}
	for _, id := range ids[:2] {
		env.Client().Do("GET", fmt.Sprintf("/api/restaurants/%d", id), nil).ExpectError(http.StatusNotFound, "not_found")
	}
	env.Client().Do("GET", fmt.Sprintf("/api/restaurants/%d", ids[2]), nil).Expect(http.StatusOK)
	if env.Images.Len() != photos-2 {
		t.Errorf("photos = %d, want %d", env.Images.Len(), photos-2)
	}
	owner.Do("POST", "/api/admin/restaurants/delete", map[string]any{"ids": ids[2:]}).ExpectError(http.StatusForbidden, "admin_only")
}

// Admin lists take up to 500 rows per page.
func TestAdminPageSize(t *testing.T) {
	env := apitest.New(t)
	admin := env.Signup("admin@example.com", "Admin")
	env.MakeAdmin("admin@example.com")
	if _, err := env.DB.Exec(t.Context(), `
		INSERT INTO accounts (email, display_name)
		SELECT format('u%s@example.com', g), 'U' FROM generate_series(1, 600) g`); err != nil {
		t.Fatal(err)
	}
	var out struct {
		Users []struct{ ID int64 }
		Total int
	}
	admin.Do("GET", "/api/admin/users?limit=500", nil).Expect(http.StatusOK).JSON(&out)
	if len(out.Users) != 500 || out.Total != 601 {
		t.Fatalf("got %d users of %d, want 500 of 601", len(out.Users), out.Total)
	}
}
