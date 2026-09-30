package service_test

import (
	"context"
	"errors"
	"testing"

	"restaurants/internal/auth"
	"restaurants/internal/auth/model"
	"restaurants/internal/platform/apperr"
	"restaurants/internal/platform/database"
	"restaurants/internal/testutil/testdb"
)

func TestLoginWithGoogle(t *testing.T) {
	ctx := context.Background()
	svc := auth.NewService(database.New(testdb.New(t)))

	t.Run("new account", func(t *testing.T) {
		a, err := svc.LoginWithGoogle(ctx, model.GoogleIdentity{Subject: "g-new", Email: "New@Example.com", EmailVerified: true, Name: "New Person"})
		if err != nil {
			t.Fatal(err)
		}
		if a.Email != "new@example.com" || !a.EmailVerified || a.HasPassword || a.DisplayName != "New Person" {
			t.Fatalf("account = %+v", a)
		}
		again, err := svc.LoginWithGoogle(ctx, model.GoogleIdentity{Subject: "g-new", Email: "new@example.com", EmailVerified: true})
		if err != nil || again.ID != a.ID {
			t.Fatalf("second login = %+v, %v; want same account", again, err)
		}
	})

	t.Run("unverified Google email is rejected", func(t *testing.T) {
		_, err := svc.LoginWithGoogle(ctx, model.GoogleIdentity{Subject: "g-unv", Email: "unv@example.com"})
		if !errors.Is(err, model.ErrGoogleEmailUnverified) {
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
		sess, err := svc.CreateSession(ctx, squatter.ID)
		if err != nil {
			t.Fatal(err)
		}

		victim, err := svc.LoginWithGoogle(ctx, model.GoogleIdentity{Subject: "g-victim", Email: "victim@example.com", EmailVerified: true})
		if err != nil {
			t.Fatal(err)
		}
		if victim.ID != squatter.ID || victim.HasPassword || !victim.EmailVerified {
			t.Fatalf("linked account = %+v", victim)
		}
		if _, err := svc.Login(ctx, "victim@example.com", "attacker-pw"); err == nil {
			t.Fatal("attacker password still works")
		}
		if _, err := svc.AccountForSession(ctx, sess.Token); !errors.Is(err, model.ErrNoSession) {
			t.Fatalf("attacker session still valid: %v", err)
		}

		// The victim can now set their own password, which is trusted.
		if err := svc.SetPassword(ctx, victim.ID, "victims-own-pw"); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Login(ctx, "victim@example.com", "victims-own-pw"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("verified account keeps its password when linking", func(t *testing.T) {
		// victim@ is verified now; a later link must not wipe the password again.
		if _, err := svc.LoginWithGoogle(ctx, model.GoogleIdentity{Subject: "g-victim", Email: "victim@example.com", EmailVerified: true}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Login(ctx, "victim@example.com", "victims-own-pw"); err != nil {
			t.Fatalf("password lost on re-login: %v", err)
		}
	})

	t.Run("email already linked to another Google account", func(t *testing.T) {
		_, err := svc.LoginWithGoogle(ctx, model.GoogleIdentity{Subject: "g-other", Email: "victim@example.com", EmailVerified: true})
		if e, ok := errors.AsType[*apperr.Error](err); !ok || e.Code != "google_conflict" {
			t.Fatalf("err = %v, want google_conflict", err)
		}
	})
}
