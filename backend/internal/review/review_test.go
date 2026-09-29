package review_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"restaurants/internal/testutil/apitest"
)

type summary struct {
	Name        string   `json:"name"`
	Rating      *float64 `json:"rating"`
	ReviewCount int      `json:"review_count"`
}

func getSummary(t *testing.T, c *apitest.Client, id int64) summary {
	t.Helper()
	var s summary
	c.Do("GET", fmt.Sprintf("/api/restaurants/%d", id), nil).Expect(http.StatusOK).JSON(&s)
	return s
}

func review(rating int, body string) map[string]any {
	return map[string]any{"rating": rating, "body": body}
}

func TestReviewLifecycleAndAggregate(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	alice := env.Signup("alice@example.com", "Alice")
	bob := env.Signup("bob@example.com", "Bob")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Rated", 10))
	path := fmt.Sprintf("/api/restaurants/%d/reviews/me", id)

	if s := getSummary(t, alice, id); s.Rating != nil || s.ReviewCount != 0 {
		t.Fatalf("no reviews yet: %+v", s)
	}

	owner.Do("PUT", path, review(5, "Best place ever")).ExpectError(http.StatusForbidden, "R-REVIEW-3")
	alice.Do("PUT", path, review(6, "Too good")).ExpectError(http.StatusUnprocessableEntity, "R-REVIEW-1")
	alice.Do("PUT", path, review(4, "   ")).ExpectError(http.StatusUnprocessableEntity, "R-REVIEW-1")
	env.Client().Do("PUT", path, review(4, "anon")).ExpectError(http.StatusUnauthorized, "unauthorized")

	alice.Do("PUT", path, review(4, "Good noodles")).Expect(http.StatusCreated)
	bob.Do("PUT", path, review(5, "Great")).Expect(http.StatusCreated)
	if s := getSummary(t, alice, id); *s.Rating != 4.5 || s.ReviewCount != 2 {
		t.Fatalf("after two reviews: %+v", s)
	}

	// Writing again edits instead of adding (R-REVIEW-2).
	alice.Do("PUT", path, review(2, "Went downhill")).Expect(http.StatusOK)
	if s := getSummary(t, alice, id); *s.Rating != 3.5 || s.ReviewCount != 2 {
		t.Fatalf("after edit: %+v", s)
	}

	bob.Do("DELETE", path, nil).Expect(http.StatusNoContent)
	bob.Do("DELETE", path, nil).ExpectError(http.StatusNotFound, "not_found")
	bob.Do("GET", path, nil).ExpectError(http.StatusNotFound, "not_found")
	if s := getSummary(t, alice, id); *s.Rating != 2 || s.ReviewCount != 1 {
		t.Fatalf("after delete: %+v", s)
	}
}

func TestAverageRoundsToOneDecimal(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Round", 10))
	for i, r := range []int{5, 5, 4} { // 14/3 = 4.666…
		env.Signup(fmt.Sprintf("r%d@example.com", i), "R").
			Do("PUT", fmt.Sprintf("/api/restaurants/%d/reviews/me", id), review(r, "ok")).Expect(http.StatusCreated)
	}
	if s := getSummary(t, owner, id); *s.Rating != 4.7 {
		t.Fatalf("rating = %v, want 4.7", *s.Rating)
	}
}

func TestVerifiedAndPrivacy(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	alice := env.Signup("alice@example.com", "Alice")
	bob := env.Signup("bob@example.com", "Bob")
	id := owner.CreateRestaurant(apitest.RestaurantInput("V", 10))

	// Alice ate here yesterday (a completed reservation, inserted directly
	// because the API only books future times). Bob never booked.
	_, err := env.DB.Exec(t.Context(), `
		INSERT INTO reservations (restaurant_id, account_id, pax, starts_at, ends_at)
		SELECT $1, id, 2, now() - interval '1 day', now() - interval '23 hours'
		FROM accounts WHERE email = 'alice@example.com'`, id)
	if err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/api/restaurants/%d/reviews", id)
	alice.Do("PUT", path+"/me", review(5, "Loved it")).Expect(http.StatusCreated)
	bob.Do("PUT", path+"/me", review(3, "Heard it's ok")).Expect(http.StatusCreated)

	type list struct {
		Total   int `json:"total"`
		Reviews []struct {
			Verified bool `json:"verified"`
			Author   struct {
				DisplayName string `json:"display_name"`
				Email       string `json:"email"`
			} `json:"author"`
		} `json:"reviews"`
	}
	var pub list
	env.Client().Do("GET", path, nil).Expect(http.StatusOK).JSON(&pub)
	if pub.Total != 2 {
		t.Errorf("total = %d, want 2", pub.Total)
	}
	var one list
	env.Client().Do("GET", path+"?limit=1&offset=1", nil).Expect(http.StatusOK).JSON(&one)
	if len(one.Reviews) != 1 || one.Total != 2 {
		t.Errorf("second page = %+v", one)
	}
	verified := map[string]bool{}
	for _, r := range pub.Reviews {
		verified[r.Author.DisplayName] = r.Verified
		if r.Author.Email != "" {
			t.Errorf("public list leaked email %q", r.Author.Email)
		}
	}
	if !verified["Alice"] || verified["Bob"] {
		t.Errorf("verified = %v, want Alice only", verified)
	}

	var own list
	owner.Do("GET", path, nil).Expect(http.StatusOK).JSON(&own)
	for _, r := range own.Reviews {
		if r.Author.Email == "" {
			t.Errorf("owner should see reviewer email (R-PRIV-2): %+v", r)
		}
	}
}

// R-REVIEW-6 / ADR-0004: Highest rated uses a Bayesian average, so one
// 5-star review doesn't beat a consistently great restaurant.
func TestTopRatedIsBayesian(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	lucky := owner.CreateRestaurant(apitest.RestaurantInput("Lucky (1 × 5★)", 10))
	solid := owner.CreateRestaurant(apitest.RestaurantInput("Solid (10 × 4.8★)", 10))
	meh := owner.CreateRestaurant(apitest.RestaurantInput("Meh (10 × 3★)", 10))
	owner.CreateRestaurant(apitest.RestaurantInput("New (no reviews)", 10))

	put := func(c *apitest.Client, id int64, rating int) {
		c.Do("PUT", fmt.Sprintf("/api/restaurants/%d/reviews/me", id), review(rating, "x")).Expect(http.StatusCreated)
	}
	for i := range 10 {
		c := env.Signup(fmt.Sprintf("critic%d@example.com", i), "Critic")
		put(c, solid, map[bool]int{true: 4, false: 5}[i < 2]) // 8×5 + 2×4 = 4.8
		put(c, meh, 3)
		if i == 0 {
			put(c, lucky, 5)
		}
	}

	names := func(sort string) string {
		var out struct{ Restaurants []summary }
		env.Client().Do("GET", "/api/restaurants?sort="+sort, nil).Expect(http.StatusOK).JSON(&out)
		var n []string
		for _, r := range out.Restaurants {
			n = append(n, strings.Fields(r.Name)[0])
		}
		return strings.Join(n, ",")
	}
	// Global mean C = (5 + 48 + 30) / 21 ≈ 3.95, m = 5:
	// Solid (48+19.8)/15 ≈ 4.52 > Lucky (5+19.8)/6 ≈ 4.13 > New ≈ 3.95 > Meh ≈ 3.32
	if got := names("top_rated"); got != "Solid,Lucky,New,Meh" {
		t.Errorf("top_rated = %s", got)
	}
	if got := names("most_reviewed"); got != "Meh,Solid,Lucky,New" {
		t.Errorf("most_reviewed = %s", got)
	}
}

// R-REVIEW-8: the list reports how many reviews have each star rating, can
// show one rating only, and sorts newest or oldest first.
func TestReviewFilterSortAndCounts(t *testing.T) {
	env := apitest.New(t)
	owner := env.Signup("owner@example.com", "Owner")
	id := owner.CreateRestaurant(apitest.RestaurantInput("Rated", 10))
	stars := []int{5, 4, 5, 1, 5}
	for i, r := range stars {
		env.Signup(fmt.Sprintf("r%d@example.com", i), fmt.Sprintf("R%d", i)).
			Do("PUT", fmt.Sprintf("/api/restaurants/%d/reviews/me", id), review(r, fmt.Sprintf("review %d", i))).Expect(http.StatusCreated)
	}
	type page struct {
		Reviews []struct {
			Rating int
			Body   string
		}
		Total        int
		RatingCounts map[string]int `json:"rating_counts"`
	}
	get := func(q string) page {
		t.Helper()
		var p page
		owner.Do("GET", fmt.Sprintf("/api/restaurants/%d/reviews?%s", id, q), nil).Expect(http.StatusOK).JSON(&p)
		return p
	}

	all := get("")
	if all.Total != 5 || fmt.Sprint(all.RatingCounts) != "map[1:1 2:0 3:0 4:1 5:3]" {
		t.Fatalf("all: total %d, counts %v", all.Total, all.RatingCounts)
	}
	if all.Reviews[0].Body != "review 4" {
		t.Errorf("newest first: got %q", all.Reviews[0].Body)
	}

	fives := get("rating=5&sort=oldest&limit=2")
	if fives.Total != 3 || len(fives.Reviews) != 2 || fives.Reviews[0].Body != "review 0" || fives.Reviews[1].Body != "review 2" {
		t.Fatalf("5 stars oldest first: %+v", fives)
	}
	if fmt.Sprint(fives.RatingCounts) != fmt.Sprint(all.RatingCounts) {
		t.Errorf("counts shouldn't depend on the filter: %v", fives.RatingCounts)
	}
	if none := get("rating=3"); none.Total != 0 || len(none.Reviews) != 0 {
		t.Errorf("3 stars: %+v", none)
	}

	for _, q := range []string{"rating=6", "rating=0", "rating=x", "sort=best"} {
		owner.Do("GET", fmt.Sprintf("/api/restaurants/%d/reviews?%s", id, q), nil).ExpectError(http.StatusUnprocessableEntity, "invalid_input")
	}
}
