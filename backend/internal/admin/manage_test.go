package admin_test

import (
	"fmt"
	"net/http"
	"testing"

	"restaurants/internal/testutil/apitest"
)

type meBody struct {
	ID           int64  `json:"id"`
	DisplayName  string `json:"display_name"`
	IsAdmin      bool   `json:"is_admin"`
	Impersonator *struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"impersonator"`
}

func me(t *testing.T, c *apitest.Client) meBody {
	t.Helper()
	var m meBody
	c.Do("GET", "/api/me", nil).Expect(http.StatusOK).JSON(&m)
	return m
}

// R-ADMIN-7: impersonate a user, act as them, switch back.
func TestImpersonation(t *testing.T) {
	env := apitest.New(t)
	admin := env.Signup("admin@example.com", "Admin")
	env.MakeAdmin("admin@example.com")
	adminID := me(t, admin).ID
	user := env.Signup("user@example.com", "Uma")
	userID := me(t, user).ID
	env.Signup("other-admin@example.com", "Other admin")
	env.MakeAdmin("other-admin@example.com")
	otherAdminID := me(t, env.Login("other-admin@example.com")).ID

	// Only admins, never yourself or another admin, never a banned user.
	user.Do("POST", fmt.Sprintf("/api/admin/users/%d/impersonate", adminID), nil).ExpectError(http.StatusForbidden, "admin_only")
	admin.Do("POST", fmt.Sprintf("/api/admin/users/%d/impersonate", adminID), nil).ExpectError(http.StatusConflict, "R-ADMIN-7")
	admin.Do("POST", fmt.Sprintf("/api/admin/users/%d/impersonate", otherAdminID), nil).ExpectError(http.StatusConflict, "R-ADMIN-7")
	admin.Do("POST", "/api/admin/users/999999/impersonate", nil).Expect(http.StatusNotFound)
	admin.Do("POST", "/api/auth/impersonate/stop", nil).ExpectError(http.StatusConflict, "not_impersonating")

	admin.Do("POST", fmt.Sprintf("/api/admin/users/%d/impersonate", userID), nil).Expect(http.StatusNoContent)
	m := me(t, admin)
	if m.ID != userID || m.IsAdmin || m.Impersonator == nil || m.Impersonator.ID != adminID {
		t.Fatalf("while impersonating, /me = %+v", m)
	}
	// Acts as the user: owns what it creates, has no admin rights, can't change the password.
	id := admin.CreateRestaurant(apitest.RestaurantInput("Uma's", 10))
	var d struct {
		IsOwner bool `json:"is_owner"`
	}
	user.Do("GET", fmt.Sprintf("/api/restaurants/%d", id), nil).Expect(http.StatusOK).JSON(&d)
	if !d.IsOwner {
		t.Fatal("restaurant created while impersonating should belong to the user")
	}
	admin.Do("GET", "/api/admin/users", nil).ExpectError(http.StatusForbidden, "admin_only")
	admin.Do("PUT", "/api/me/password", map[string]string{"current_password": "x", "new_password": "password999"}).
		ExpectError(http.StatusForbidden, "R-ADMIN-7")
	// The user's own session is untouched.
	if me(t, user).Impersonator != nil {
		t.Fatal("the user's own session must not be marked")
	}

	admin.Do("POST", "/api/auth/impersonate/stop", nil).Expect(http.StatusOK)
	if m := me(t, admin); m.ID != adminID || !m.IsAdmin || m.Impersonator != nil {
		t.Fatalf("after stop, /me = %+v", m)
	}
	admin.Do("GET", "/api/admin/users", nil).Expect(http.StatusOK)
}

// An impersonation session ends when the admin loses their rights.
func TestImpersonationEndsWithAdminRights(t *testing.T) {
	env := apitest.New(t)
	admin := env.Signup("admin@example.com", "Admin")
	env.MakeAdmin("admin@example.com")
	userID := me(t, env.Signup("user@example.com", "Uma")).ID
	admin.Do("POST", fmt.Sprintf("/api/admin/users/%d/impersonate", userID), nil).Expect(http.StatusNoContent)
	env.DB.Exec(t.Context(), `UPDATE accounts SET is_admin = false WHERE email = 'admin@example.com'`)
	admin.Do("GET", "/api/me", nil).Expect(http.StatusUnauthorized)
}

// R-ADMIN-6: admins edit profiles and restaurants they don't own.
func TestAdminEditsProfileAndRestaurant(t *testing.T) {
	env := apitest.New(t)
	admin := env.Signup("admin@example.com", "Admin")
	env.MakeAdmin("admin@example.com")
	owner := env.Signup("owner@example.com", "Olivia")
	ownerID := me(t, owner).ID
	stranger := env.Signup("stranger@example.com", "Sam")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Olivia's", 10))

	stranger.Do("PUT", fmt.Sprintf("/api/admin/users/%d", ownerID), map[string]string{"display_name": "x"}).ExpectError(http.StatusForbidden, "admin_only")
	admin.Do("PUT", fmt.Sprintf("/api/admin/users/%d", ownerID), map[string]string{"display_name": "  "}).Expect(http.StatusUnprocessableEntity)
	admin.Do("PUT", "/api/admin/users/999999", map[string]string{"display_name": "Nobody"}).Expect(http.StatusNotFound)
	admin.Do("PUT", fmt.Sprintf("/api/admin/users/%d", ownerID), map[string]string{"display_name": "Olivia O."}).Expect(http.StatusOK)
	if got := me(t, owner).DisplayName; got != "Olivia O." {
		t.Fatalf("display name = %q", got)
	}

	var d struct {
		Seats     int  `json:"seats"`
		IsOwner   bool `json:"is_owner"`
		CanManage bool `json:"can_manage"`
		Upcoming  *int `json:"upcoming_reservations"`
	}
	admin.Do("GET", fmt.Sprintf("/api/restaurants/%d", id), nil).Expect(http.StatusOK).JSON(&d)
	if d.IsOwner || !d.CanManage || d.Upcoming == nil {
		t.Fatalf("admin view = %+v", d)
	}
	stranger.Do("PUT", fmt.Sprintf("/api/restaurants/%d", id), apitest.RestaurantInput("Mine now", 3)).ExpectError(http.StatusForbidden, "R-REST-2")
	admin.Do("PUT", fmt.Sprintf("/api/restaurants/%d", id), apitest.RestaurantInput("Olivia's", 12)).Expect(http.StatusOK).JSON(&d)
	if d.Seats != 12 {
		t.Fatalf("seats = %d", d.Seats)
	}
	stranger.Do("GET", fmt.Sprintf("/api/restaurants/%d/reservations", id), nil).Expect(http.StatusForbidden)
	admin.Do("GET", fmt.Sprintf("/api/restaurants/%d/reservations", id), nil).Expect(http.StatusOK)
	admin.Do("DELETE", fmt.Sprintf("/api/restaurants/%d", id), nil).Expect(http.StatusNoContent)
	owner.Do("GET", fmt.Sprintf("/api/restaurants/%d", id), nil).Expect(http.StatusNotFound)
}
