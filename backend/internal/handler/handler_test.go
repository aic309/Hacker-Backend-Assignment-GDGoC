package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"movie-watchlist-api/internal/auth"
	"movie-watchlist-api/internal/database"
	"movie-watchlist-api/internal/gemini"
	"movie-watchlist-api/internal/handler"
	"movie-watchlist-api/internal/repository"
)

type testEnv struct {
	t      *testing.T
	server *httptest.Server

	geminiAnswers []string
	geminiCalls   int
}

func newEnv(t *testing.T) *testEnv {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set, skipping database tests")
	}
	auth.Cost = bcrypt.MinCost

	ctx := context.Background()
	pool, err := database.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS watched, watchlist, users, movies CASCADE`); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO movies (id, title, year, director, genres, rating, duration, synopsis, poster_url, cast_members) VALUES
		(1, 'Inception', 2010, 'Christopher Nolan', 'Sci-Fi, Action', 8.8, '148 min', 's1', 'p1', 'A, B'),
		(2, 'Parasite', 2019, 'Bong Joon-ho', 'Drama, Thriller', 8.5, '132 min', 's2', 'p2', 'C, D'),
		(3, 'Action Comedy Show', 2000, 'Someone', 'Action Comedy', 6.0, '90 min', 's3', 'p3', 'E')`); err != nil {
		t.Fatal(err)
	}

	env := &testEnv{t: t}

	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env.geminiCalls++
		i := env.geminiCalls - 1
		if i >= len(env.geminiAnswers) {
			i = len(env.geminiAnswers) - 1
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"candidates": []any{map[string]any{
			"content": map[string]any{"parts": []any{map[string]any{"text": env.geminiAnswers[i]}}},
		}}})
	}))
	t.Cleanup(fake.Close)
	gem := gemini.New("test-key", "test-model")
	gem.SetBaseURL(fake.URL)

	h := &handler.Handler{
		Users:     repository.NewUserRepo(pool),
		Movies:    repository.NewMovieRepo(pool),
		Watchlist: repository.NewWatchlistRepo(pool),
		Watched:   repository.NewWatchedRepo(pool),
		Tokens:    auth.NewTokenManager("test-secret-test-secret-test-secret", time.Minute),
		Gemini:    gem,
	}
	env.server = httptest.NewServer(h.Routes([]string{"*"}))
	t.Cleanup(env.server.Close)
	return env
}

func (e *testEnv) call(method, path, token, body string) (int, map[string]any, []any) {
	e.t.Helper()
	var rd *bytes.Reader
	if body != "" {
		rd = bytes.NewReader([]byte(body))
	} else {
		rd = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, e.server.URL+path, rd)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	var obj map[string]any
	var arr []any
	if strings.HasPrefix(strings.TrimSpace(buf.String()), "[") {
		_ = json.Unmarshal(buf.Bytes(), &arr)
	} else {
		_ = json.Unmarshal(buf.Bytes(), &obj)
	}
	return resp.StatusCode, obj, arr
}

func (e *testEnv) expectStatus(want int, method, path, token, body string) map[string]any {
	e.t.Helper()
	got, obj, _ := e.call(method, path, token, body)
	if got != want {
		e.t.Fatalf("%s %s = %d, want %d (body: %v)", method, path, got, want, obj)
	}
	return obj
}

func (e *testEnv) login(email string) string {
	e.t.Helper()
	body := `{"email":"` + email + `","password":"password123","username":"u"}`
	e.call("POST", "/api/auth/register", "", body)
	obj := e.expectStatus(200, "POST", "/api/auth/login", "", body)
	return obj["token"].(string)
}

func TestRegisterAndLogin(t *testing.T) {
	e := newEnv(t)

	obj := e.expectStatus(201, "POST", "/api/auth/register", "", `{"email":"a@example.com","password":"password123","username":"alice"}`)
	user := obj["user"].(map[string]any)
	if user["email"] != "a@example.com" || user["username"] != "alice" {
		t.Errorf("user = %v", user)
	}
	if _, leaked := user["password_hash"]; leaked {
		t.Error("password hash must never be sent")
	}

	obj = e.expectStatus(201, "POST", "/api/auth/register", "", `{"email":"bob@example.com","password":"password123"}`)
	if obj["user"].(map[string]any)["username"] != "bob" {
		t.Errorf("default username = %v", obj["user"])
	}

	e.expectStatus(409, "POST", "/api/auth/register", "", `{"email":"A@Example.com","password":"password123"}`)

	for name, body := range map[string]string{
		"bad email":       `{"email":"nope","password":"password123"}`,
		"email with name": `{"email":"Al <a@b.com>","password":"password123"}`,
		"short password":  `{"email":"c@example.com","password":"short"}`,
		"long password":   `{"email":"c@example.com","password":"` + strings.Repeat("x", 73) + `"}`,
		"long username":   `{"email":"c@example.com","password":"password123","username":"` + strings.Repeat("x", 51) + `"}`,
		"not json":        `{"email":`,
	} {
		t.Run(name, func(t *testing.T) { e.expectStatus(400, "POST", "/api/auth/register", "", body) })
	}

	obj = e.expectStatus(200, "POST", "/api/auth/login", "", `{"email":"a@example.com","password":"password123"}`)
	if obj["token"] == "" || obj["user"].(map[string]any)["username"] != "alice" {
		t.Errorf("login reply = %v", obj)
	}

	wrong := e.expectStatus(401, "POST", "/api/auth/login", "", `{"email":"a@example.com","password":"wrong-password"}`)
	unknown := e.expectStatus(401, "POST", "/api/auth/login", "", `{"email":"ghost@example.com","password":"password123"}`)
	if wrong["error"] != unknown["error"] {
		t.Errorf("messages differ: %q vs %q (this would reveal which emails exist)", wrong["error"], unknown["error"])
	}
	e.expectStatus(400, "POST", "/api/auth/login", "", `{"email":"","password":""}`)
}

func TestPasswordIsStoredHashed(t *testing.T) {
	e := newEnv(t)
	e.login("hash@example.com")
	pool, _ := database.Connect(context.Background(), os.Getenv("TEST_DATABASE_URL"))
	defer pool.Close()
	var stored string
	if err := pool.QueryRow(context.Background(), `SELECT password_hash FROM users WHERE email='hash@example.com'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == "password123" || !strings.HasPrefix(stored, "$2") {
		t.Errorf("stored value %q is not a bcrypt hash", stored)
	}
}

func TestMovies(t *testing.T) {
	e := newEnv(t)

	_, _, all := e.call("GET", "/api/movies", "", "")
	if len(all) != 3 {
		t.Fatalf("got %d movies, want 3", len(all))
	}
	first := all[0].(map[string]any)
	if first["title"] != "Inception" || first["cast_members"] != "A, B" {
		t.Errorf("first movie = %v", first)
	}
	if genres := first["genres"].([]any); len(genres) != 2 || genres[0] != "Sci-Fi" {
		t.Errorf("genres = %v", first["genres"])
	}

	count := func(query string) int {
		_, _, list := e.call("GET", "/api/movies?"+query, "", "")
		return len(list)
	}
	if n := count("genre=Action"); n != 1 {
		t.Errorf("genre=Action -> %d, want 1", n)
	}
	if n := count("genre=action"); n != 1 {
		t.Errorf("genre is case-insensitive, got %d", n)
	}
	if n := count("search=nolan"); n != 1 {
		t.Errorf("search=nolan -> %d, want 1", n)
	}
	if n := count("search=%25"); n != 0 {
		t.Errorf("search=%% -> %d, want 0", n)
	}
	if n := count("search=x'%3B+DROP+TABLE+movies%3B--"); n != 0 {
		t.Errorf("injection text -> %d, want 0", n)
	}
	if n := count("shuffle=true"); n != 3 {
		t.Errorf("shuffle -> %d, want 3", n)
	}
	e.expectStatus(200, "GET", "/api/movies/1", "", "")
	e.expectStatus(404, "GET", "/api/movies/999", "", "")
	e.expectStatus(400, "GET", "/api/movies/abc", "", "")
	e.expectStatus(400, "GET", "/api/movies/0", "", "")
}

func TestProtectedRoutesNeedAToken(t *testing.T) {
	e := newEnv(t)
	for _, rt := range []struct{ method, path, body string }{
		{"GET", "/api/watchlist", ""},
		{"POST", "/api/watchlist", `{"movie_id":1}`},
		{"DELETE", "/api/watchlist/1", ""},
		{"GET", "/api/watched", ""},
		{"POST", "/api/watched", `{"movie_id":1,"rating":5}`},
	} {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			e.expectStatus(401, rt.method, rt.path, "", rt.body)
			e.expectStatus(401, rt.method, rt.path, "garbage.token.here", rt.body)
		})
	}
}

func TestWatchlist(t *testing.T) {
	e := newEnv(t)
	tok := e.login("w@example.com")

	obj := e.expectStatus(201, "POST", "/api/watchlist", tok, `{"movieId":1,"movie_id":1}`)
	if obj["movie_id"].(float64) != 1 || obj["movie"].(map[string]any)["title"] != "Inception" {
		t.Errorf("reply = %v", obj)
	}
	e.expectStatus(409, "POST", "/api/watchlist", tok, `{"movie_id":1}`)
	e.expectStatus(201, "POST", "/api/watchlist", tok, `{"movieId":"2"}`)
	e.expectStatus(404, "POST", "/api/watchlist", tok, `{"movie_id":999}`)
	e.expectStatus(400, "POST", "/api/watchlist", tok, `{}`)
	e.expectStatus(400, "POST", "/api/watchlist", tok, `{"movie_id":"abc"}`)
	e.expectStatus(400, "POST", "/api/watchlist", tok, `{"movie_id":0}`)

	_, _, list := e.call("GET", "/api/watchlist", tok, "")
	if len(list) != 2 {
		t.Fatalf("watchlist has %d items, want 2", len(list))
	}

	e.expectStatus(204, "DELETE", "/api/watchlist/1", tok, "")
	e.expectStatus(404, "DELETE", "/api/watchlist/1", tok, "")
	e.expectStatus(400, "DELETE", "/api/watchlist/abc", tok, "")
	_, _, list = e.call("GET", "/api/watchlist", tok, "")
	if len(list) != 1 {
		t.Errorf("watchlist has %d items after delete, want 1", len(list))
	}
}

func TestWatched(t *testing.T) {
	e := newEnv(t)
	tok := e.login("v@example.com")

	obj := e.expectStatus(201, "POST", "/api/watched", tok, `{"movieId":1,"movie_id":1,"rating":5,"review":"great","opinion":"great"}`)
	if obj["rating"].(float64) != 5 || obj["review"] != "great" || obj["movie"].(map[string]any)["id"].(float64) != 1 {
		t.Errorf("reply = %v", obj)
	}
	e.expectStatus(409, "POST", "/api/watched", tok, `{"movie_id":1,"rating":3}`)
	e.expectStatus(404, "POST", "/api/watched", tok, `{"movie_id":999,"rating":3}`)

	for name, body := range map[string]string{
		"rating 0":      `{"movie_id":2,"rating":0}`,
		"rating 6":      `{"movie_id":2,"rating":6}`,
		"rating -1":     `{"movie_id":2,"rating":-1}`,
		"rating 2.5":    `{"movie_id":2,"rating":2.5}`,
		"rating text":   `{"movie_id":2,"rating":"abc"}`,
		"rating absent": `{"movie_id":2}`,
		"movie absent":  `{"rating":3}`,
		"long review":   `{"movie_id":2,"rating":3,"review":"` + strings.Repeat("x", 2001) + `"}`,
	} {
		t.Run(name, func(t *testing.T) { e.expectStatus(400, "POST", "/api/watched", tok, body) })
	}

	obj = e.expectStatus(201, "POST", "/api/watched", tok, `{"movie_id":2,"rating":"4"}`)
	if obj["review"] != "" {
		t.Errorf("review = %v", obj["review"])
	}
	_, _, list := e.call("GET", "/api/watched", tok, "")
	if len(list) != 2 {
		t.Errorf("watched has %d items, want 2", len(list))
	}
}

func TestUsersCannotSeeEachOthersData(t *testing.T) {
	e := newEnv(t)
	alice := e.login("alice@example.com")
	bob := e.login("bob@example.com")

	e.expectStatus(201, "POST", "/api/watchlist", alice, `{"movie_id":1}`)
	e.expectStatus(201, "POST", "/api/watched", alice, `{"movie_id":1,"rating":5,"review":"mine"}`)

	_, _, list := e.call("GET", "/api/watchlist", bob, "")
	if len(list) != 0 {
		t.Error("bob can see alice's watchlist")
	}
	_, _, list = e.call("GET", "/api/watched", bob, "")
	if len(list) != 0 {
		t.Error("bob can see alice's watched list")
	}

	e.expectStatus(404, "DELETE", "/api/watchlist/1", bob, "")

	_, _, list = e.call("GET", "/api/watchlist", alice, "")
	if len(list) != 1 {
		t.Error("alice's item disappeared")
	}

	e.expectStatus(201, "POST", "/api/watchlist", bob, `{"movie_id":1}`)
}

func TestRecommend(t *testing.T) {
	t.Run("returns a movie from the database", func(t *testing.T) {
		e := newEnv(t)
		e.geminiAnswers = []string{`{"movie_id": 2, "reasoning": "Cocok untukmu."}`}
		obj := e.expectStatus(200, "POST", "/api/movies/recommend", "", `{"prompt":"film thriller korea"}`)
		if obj["movie_id"].(float64) != 2 || obj["reasoning"] != "Cocok untukmu." {
			t.Errorf("reply = %v", obj)
		}
		if obj["movie"].(map[string]any)["title"] != "Parasite" {
			t.Errorf("movie = %v", obj["movie"])
		}
	})
	t.Run("a made-up movie never reaches the user", func(t *testing.T) {
		e := newEnv(t)
		e.geminiAnswers = []string{`{"movie_id": 777, "reasoning": "x"}`}
		e.expectStatus(502, "POST", "/api/movies/recommend", "", `{"prompt":"anything"}`)
		if e.geminiCalls != 2 {
			t.Errorf("gemini calls = %d, want 2 (first try + one retry)", e.geminiCalls)
		}
	})
	t.Run("retry fixes a wrong first answer", func(t *testing.T) {
		e := newEnv(t)
		e.geminiAnswers = []string{`{"movie_id": 777, "reasoning": "x"}`, `{"movie_id": 1, "reasoning": "ok"}`}
		obj := e.expectStatus(200, "POST", "/api/movies/recommend", "", `{"prompt":"anything"}`)
		if obj["movie_id"].(float64) != 1 {
			t.Errorf("reply = %v", obj)
		}
	})
	t.Run("input errors", func(t *testing.T) {
		e := newEnv(t)
		e.geminiAnswers = []string{`{"movie_id": 1, "reasoning": "ok"}`}
		e.expectStatus(400, "POST", "/api/movies/recommend", "", `{}`)
		e.expectStatus(400, "POST", "/api/movies/recommend", "", `{"prompt":"   "}`)
		e.expectStatus(400, "POST", "/api/movies/recommend", "", `{"prompt":"`+strings.Repeat("x", 501)+`"}`)
		if e.geminiCalls != 0 {
			t.Error("Gemini must not be called for bad input")
		}

		e.expectStatus(200, "POST", "/api/movies/recommend", "", `{"query":"horror"}`)
	})
}

func TestUnknownRouteIsJSON404(t *testing.T) {
	e := newEnv(t)
	obj := e.expectStatus(404, "GET", "/api/nothing", "", "")
	if obj["error"] == nil {
		t.Error("404 should be JSON with an error field")
	}
	e.expectStatus(200, "GET", "/api/health", "", "")
	_ = strconv.Itoa
}
