package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"restaurants/internal/httpx"
	"restaurants/internal/testdb"
)

func TestLoginWithGoogle(t *testing.T) {
	ctx := context.Background()
	svc := NewService(testdb.New(t))

	t.Run("new account", func(t *testing.T) {
		a, err := svc.LoginWithGoogle(ctx, GoogleIdentity{Subject: "g-new", Email: "New@Example.com", EmailVerified: true, Name: "New Person"})
		if err != nil {
			t.Fatal(err)
		}
		if a.Email != "new@example.com" || !a.EmailVerified || a.HasPassword || a.DisplayName != "New Person" {
			t.Fatalf("account = %+v", a)
		}
		again, err := svc.LoginWithGoogle(ctx, GoogleIdentity{Subject: "g-new", Email: "new@example.com", EmailVerified: true})
		if err != nil || again.ID != a.ID {
			t.Fatalf("second login = %+v, %v; want same account", again, err)
		}
	})

	t.Run("unverified Google email is rejected", func(t *testing.T) {
		_, err := svc.LoginWithGoogle(ctx, GoogleIdentity{Subject: "g-unv", Email: "unv@example.com"})
		if !errors.Is(err, ErrGoogleEmailUnverified) {
			t.Fatalf("err = %v", err)
		}
	})

	// ADR-0002 pre-hijack: an attacker registers the victim's email with a
	// password first. When the victim signs in with Google, the attacker's
	// password and sessions must stop working.
	t.Run("linking drops an unverified password and its sessions", func(t *testing.T) {
		squatter, err := svc.Signup(ctx, "victim@example.com", "attacker-pw", "Squatter")
		if err != nil {
			t.Fatal(err)
		}
		token, _, err := svc.CreateSession(ctx, squatter.ID)
		if err != nil {
			t.Fatal(err)
		}

		victim, err := svc.LoginWithGoogle(ctx, GoogleIdentity{Subject: "g-victim", Email: "victim@example.com", EmailVerified: true})
		if err != nil {
			t.Fatal(err)
		}
		if victim.ID != squatter.ID || victim.HasPassword || !victim.EmailVerified {
			t.Fatalf("linked account = %+v", victim)
		}
		if _, err := svc.Login(ctx, "victim@example.com", "attacker-pw"); err == nil {
			t.Fatal("attacker password still works")
		}
		if _, err := svc.AccountForSession(ctx, token); !errors.Is(err, ErrNoSession) {
			t.Fatalf("attacker session still valid: %v", err)
		}

		// The victim can now set their own password, which is trusted.
		if err := svc.SetPassword(ctx, victim.ID, "", "victims-own-pw"); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Login(ctx, "victim@example.com", "victims-own-pw"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("verified account keeps its password when linking", func(t *testing.T) {
		// victim@ is verified now; a later link must not wipe the password again.
		if _, err := svc.LoginWithGoogle(ctx, GoogleIdentity{Subject: "g-victim", Email: "victim@example.com", EmailVerified: true}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Login(ctx, "victim@example.com", "victims-own-pw"); err != nil {
			t.Fatalf("password lost on re-login: %v", err)
		}
	})

	t.Run("email already linked to another Google account", func(t *testing.T) {
		_, err := svc.LoginWithGoogle(ctx, GoogleIdentity{Subject: "g-other", Email: "victim@example.com", EmailVerified: true})
		if e, ok := errors.AsType[*httpx.Error](err); !ok || e.Code != "google_conflict" {
			t.Fatalf("err = %v, want google_conflict", err)
		}
	})
}

func TestSafeNext(t *testing.T) {
	for in, want := range map[string]string{
		"":                     "/",
		"/me/reservations":     "/me/reservations",
		"//evil.example.com":   "/",
		"https://evil.example": "/",
		`/\evil.example.com`:   "/",
		"relative":             "/",
	} {
		if got := safeNext(in); got != want {
			t.Errorf("safeNext(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGoogleCallbackRejectsBadState(t *testing.T) {
	h := NewHandler(nil, false, GoogleConfig{ClientID: "test"})
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/api/auth/google/callback?state=forged&code=x", nil)
	req.AddCookie(&http.Cookie{Name: googleFlowCookie, Value: "real-state|n|v|/"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/login?error=google_state" {
		t.Fatalf("got %d → %q", rec.Code, rec.Header().Get("Location"))
	}

	disabled := http.NewServeMux()
	NewHandler(nil, false, GoogleConfig{}).Register(disabled)
	rec = httptest.NewRecorder()
	disabled.ServeHTTP(rec, httptest.NewRequest("GET", "/api/auth/google/start", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("disabled start = %d, want 404", rec.Code)
	}
}
