package auth_test

import (
	"net/http"
	"testing"

	"restaurants/internal/testutil/apitest"
)

func TestSignupLoginLogout(t *testing.T) {
	env := apitest.New(t)

	alice := env.Signup("Alice@Example.com", "Alice")
	var me struct {
		Email       string `json:"email"`
		HasPassword bool   `json:"has_password"`
	}
	alice.Do("GET", "/api/me", nil).Expect(http.StatusOK).JSON(&me)
	if me.Email != "alice@example.com" || !me.HasPassword {
		t.Fatalf("me = %+v, want normalized email with password", me)
	}

	alice.Do("POST", "/api/auth/logout", nil).Expect(http.StatusNoContent)
	alice.Do("GET", "/api/me", nil).ExpectError(http.StatusUnauthorized, "unauthorized")

	alice.Do("POST", "/api/auth/login", map[string]string{"email": "alice@example.com", "password": "wrong-password"}).
		ExpectError(http.StatusUnauthorized, "invalid_credentials")
	alice.Do("POST", "/api/auth/login", map[string]string{"email": " ALICE@example.com ", "password": "password123"}).
		Expect(http.StatusOK)
	alice.Do("GET", "/api/me", nil).Expect(http.StatusOK)
}

func TestSignupValidation(t *testing.T) {
	env := apitest.New(t)
	env.Signup("bob@example.com", "Bob")
	c := env.Client()

	c.Do("POST", "/api/auth/signup", map[string]string{"email": "BOB@example.com", "password": "password123", "display_name": "B"}).
		ExpectError(http.StatusConflict, "email_taken")
	c.Do("POST", "/api/auth/signup", map[string]string{"email": "not-an-email", "password": "password123", "display_name": "X"}).
		ExpectError(http.StatusUnprocessableEntity, "invalid_input")
	c.Do("POST", "/api/auth/signup", map[string]string{"email": "x@example.com", "password": "short", "display_name": "X"}).
		ExpectError(http.StatusUnprocessableEntity, "invalid_input")
	c.Do("POST", "/api/auth/signup", map[string]any{"email": "x@example.com", "password": "password123", "display_name": "X", "admin": true}).
		ExpectError(http.StatusBadRequest, "bad_request")
}

func TestChangePassword(t *testing.T) {
	env := apitest.New(t)
	carol := env.Signup("carol@example.com", "Carol")

	carol.Do("PUT", "/api/me/password", map[string]string{"current_password": "nope-nope", "new_password": "new-password"}).
		ExpectError(http.StatusUnauthorized, "invalid_credentials")
	carol.Do("PUT", "/api/me/password", map[string]string{"current_password": "password123", "new_password": "new-password"}).
		Expect(http.StatusNoContent)

	env.Client().Do("POST", "/api/auth/login", map[string]string{"email": "carol@example.com", "password": "new-password"}).
		Expect(http.StatusOK)
}

func TestForgedCookieIsAnonymous(t *testing.T) {
	env := apitest.New(t)
	req, _ := http.NewRequest("GET", "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "forged"})
	env.Client().Send(req).ExpectError(http.StatusUnauthorized, "unauthorized")
}

func TestUpdateProfile(t *testing.T) {
	env := apitest.New(t)
	dana := env.Signup("dana@example.com", "Dana")

	var me struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
	}
	dana.Do("PUT", "/api/me", map[string]string{"display_name": "  Dana K.  "}).Expect(http.StatusOK).JSON(&me)
	if me.DisplayName != "Dana K." || me.Email != "dana@example.com" {
		t.Fatalf("me = %+v", me)
	}
	dana.Do("GET", "/api/me", nil).Expect(http.StatusOK).JSON(&me)
	if me.DisplayName != "Dana K." {
		t.Fatalf("rename not saved: %+v", me)
	}
	dana.Do("PUT", "/api/me", map[string]string{"display_name": "   "}).ExpectError(http.StatusUnprocessableEntity, "invalid_input")
	dana.Do("PUT", "/api/me", map[string]string{"email": "x@example.com"}).ExpectError(http.StatusBadRequest, "bad_request")
	env.Client().Do("PUT", "/api/me", map[string]string{"display_name": "X"}).ExpectError(http.StatusUnauthorized, "unauthorized")
}
