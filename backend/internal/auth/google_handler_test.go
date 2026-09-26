package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"restaurants/internal/platform/config"
)

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
	gin.SetMode(gin.TestMode)
	mux := gin.New()
	RegisterRoutes(mux.Group("/api"), NewHandler(nil, false, config.Google{ClientID: "test"}))

	req := httptest.NewRequest("GET", "/api/auth/google/callback?state=forged&code=x", nil)
	req.AddCookie(&http.Cookie{Name: googleFlowCookie, Value: "real-state|n|v|/"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/login?error=google_state" {
		t.Fatalf("got %d → %q", rec.Code, rec.Header().Get("Location"))
	}

	disabled := gin.New()
	RegisterRoutes(disabled.Group("/api"), NewHandler(nil, false, config.Google{}))
	rec = httptest.NewRecorder()
	disabled.ServeHTTP(rec, httptest.NewRequest("GET", "/api/auth/google/start", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("disabled start = %d, want 404", rec.Code)
	}
}
