# Movie Watchlist API (Go + PostgreSQL)

REST API for a movie watchlist and review app (GDGoC UGM Hacker – Back-End task).
Users register and log in, browse 20 movies, keep a watchlist, log watched movies with a 1–5 rating and
review, and ask **Google Gemini** for a recommendation that can only be a movie from our own database.

Built with Go (standard library `net/http`), PostgreSQL (`pgx`), bcrypt and JWT.
It works with the frontend of the starter repo without changes.

## Setup

**You need:** Go 1.23+, PostgreSQL 14+ (running), and a free Gemini API key from https://aistudio.google.com

```bash
# 1. create an empty database (once)
createdb moviedb            # or, in psql:  CREATE DATABASE moviedb;

# 2. settings
cp .env.example .env        # then edit .env (see the table below)
openssl rand -hex 32        # use this output as JWT_SECRET

# 3. download the libraries
go mod tidy

# 4. load the 20 movies (safe to run many times)
go run ./cmd/seed

# 5. start the server
go run ./cmd/api            # http://localhost:8000
```

The tables are created automatically when the server (or the seed command) starts.
`go run ./cmd/seed path/to/movies.json` loads another file; by default it reads `data/movies.json`.

### Environment variables (`.env`)

| Name | Required | Meaning |
|---|---|---|
| `JWT_SECRET` | yes | Secret that signs the tokens. At least 32 characters, or the server refuses to start. |
| `DATABASE_URL` | yes | `postgres://USER:PASSWORD@HOST:PORT/DBNAME?sslmode=disable`. Default `postgres://localhost:5432/moviedb?sslmode=disable` |
| `GEMINI_API_KEY` | for `/recommend` | Key from Google AI Studio. Without it, `/recommend` answers 503 and everything else still works. |
| `GEMINI_MODEL` | no | Default `gemini-3.8-flash`. Change it if Google renames or retires a model. |
| `JWT_EXPIRE_MINUTES` | no | Default `60` |
| `PORT` | no | Default `8000` |
| `CORS_ORIGINS` | no | Frontend addresses allowed to call the API, comma separated. Default `*` |

### Run the frontend (optional)

The starter repo has a ready frontend. In another terminal:

```bash
cd frontend
python3 -m http.server 3000      # open http://localhost:3000
```

It calls `http://localhost:8000/api` by default.

### Run the tests

```bash
go test ./...
```
Unit tests (JWT, bcrypt, middleware, Gemini grounding with a fake Gemini server) always run.
The API tests use a real PostgreSQL and need an **empty test database**. Their tables are dropped
at the start of every test, so never use your real database:

```bash
createdb moviedb_test
TEST_DATABASE_URL="postgres://localhost:5432/moviedb_test?sslmode=disable" go test ./...
```
No API key or internet is needed: Gemini is replaced by a fake server.

## Endpoints

Base URL `http://localhost:8000/api`. Protected routes need `Authorization: Bearer <token>`.
Errors always look like `{"error": "message"}`.

| Method | URL | Auth | Body / query | Success |
|---|---|---|---|---|
| POST | `/auth/register` | – | `{email, password, username?}` | 201 `{message, user}` |
| POST | `/auth/login` | – | `{email, password}` | 200 `{token, user}` |
| GET | `/movies` | – | `?genre=Action&search=nolan&shuffle=true` | 200 `[movie]` |
| GET | `/movies/{id}` | – | | 200 `movie` |
| POST | `/movies/recommend` | – | `{prompt}` | 200 `{movie_id, reasoning, movie}` |
| GET | `/watchlist` | Bearer | | 200 `[{id, movie_id, movie, created_at}]` |
| POST | `/watchlist` | Bearer | `{movie_id}` (`movieId` also works) | 201 item |
| DELETE | `/watchlist/{movieId}` | Bearer | | 204 |
| GET | `/watched` | Bearer | | 200 `[{id, movie_id, rating, review, movie, created_at}]` |
| POST | `/watched` | Bearer | `{movie_id, rating 1-5, review?}` | 201 item |
| GET | `/health` | – | | 200 `{status: "ok"}` |

### Status codes

| Code | When |
|---|---|
| 400 | Bad input: invalid email, password shorter than 8 or longer than 72 bytes, rating outside 1–5, empty prompt, broken JSON, id that is not a number |
| 401 | No / invalid / expired token; wrong email or password |
| 404 | Movie not found; movie not in your watchlist; unknown URL |
| 409 | Email already registered; movie already in your watchlist; movie already logged as watched |
| 413 | Request body over 1 MB |
| 500 | Unexpected server error (details go to the log, not to the client) |
| 502 | Gemini failed, or kept answering with a movie that is not in the database |
| 503 | `GEMINI_API_KEY` is not set |

## Database design (ERD)

```
users ──< watchlist >── movies
users ──< watched   >── movies
```

| Table | Columns |
|---|---|
| `movies` | `id` PK, `title`, `year`, `director`, `genres`, `rating`, `duration`, `synopsis`, `poster_url`, `cast_members` |
| `users` | `id` PK, `username`, `email` **UNIQUE**, `password_hash`, `created_at` |
| `watchlist` | `id` PK, `user_id` FK, `movie_id` FK, `created_at`, **UNIQUE(user_id, movie_id)** |
| `watched` | `id` PK, `user_id` FK, `movie_id` FK, `rating`, `review`, `created_at`, **UNIQUE(user_id, movie_id)**, **CHECK(rating BETWEEN 1 AND 5)** |

Why it is built this way:

- `watchlist` and `watched` are **join tables**: one user has many movies and one movie has many users (many-to-many).
  They are two tables because they mean different things: "I want to watch this" vs. "I watched this and rated it".
- **The database enforces the rules, not only the code.** `UNIQUE` blocks duplicates, `CHECK` blocks ratings outside 1–5,
  foreign keys block rows for movies or users that do not exist. If the Go code had a bug, bad data still could not get in.
  The code turns these database errors into 409 / 404.
- `ON DELETE CASCADE`: when a user or movie is deleted, its watchlist and watched rows go with it (no orphan rows).
- Indexes: the `UNIQUE` on `users.email` makes login fast; the `UNIQUE(user_id, movie_id)` indexes serve "my list" queries;
  extra indexes on `movie_id` make joins and cascade deletes fast.
- `genres` and `cast_members` are comma-separated text, the same as the starter `seed.sql`, so the data and frontend stay
  compatible. The API sends genres as a list, and the cast both as a list (`cast`) and as text (`cast_members`).
- The email is stored in lowercase, so `A@x.com` and `a@x.com` are the same account.

## Security

- **Passwords:** hashed with bcrypt (random salt per password, cost 10). The plain password is never stored, logged or returned
  (`PasswordHash` has `json:"-"`). Passwords longer than 72 bytes are rejected, because bcrypt ignores everything after that.
- **JWT:** HS256, holds the user id (`sub`) and an expiry (`exp`, default 60 minutes). On every protected request the signature
  and expiry are checked, only HS256 is accepted (blocks `alg: none`), and tokens without an expiry are refused.
  `JWT_SECRET` must be at least 32 characters.
- **Login errors:** "unknown email" and "wrong password" give the same 401 message, and an unknown email still runs one bcrypt
  check, so neither the message nor the response time reveals which emails exist.
- **Data isolation:** watchlist and watched queries always use the user id **from the token**, never from the request body.
  One user can not read or delete another user's data (tested).
- **SQL injection:** every query uses placeholders (`$1`, `$2`). User text never becomes part of the SQL.
  `%` and `_` in the search are escaped so they are normal characters.
- **Input validation:** email format, password length, rating 1–5, review ≤ 2000 chars, prompt ≤ 500 chars, body ≤ 1 MB.
- **Errors:** unexpected errors are logged on the server and the client only sees "internal server error".
- **CORS:** only the origins in `CORS_ORIGINS` (default `*`, set it to your frontend address in production).
- **Timeouts:** the HTTP server has read/write/idle timeouts, so slow clients can not hold connections forever.
- **Stateless tokens:** logging out is done by the client forgetting the token; a token stays valid until it expires.

## Gemini grounding (`internal/gemini`)

The goal: the AI may only recommend movies that exist in **our** database.

1. **The catalog is in the prompt.** Every request loads all movies from PostgreSQL and lists them
   (`id | title (year) | genres | director | rating | short synopsis`).
2. **System instruction:** recommend exactly one movie, only from the catalog, answer with its id, never mention other movies.
3. **Forced JSON answer** (`responseSchema`): `{"movie_id": <int>, "reasoning": "<text>"}`. No free text to parse.
4. **We verify the id.** `movie_id` must be one of the ids we sent. If not, Gemini gets **one** retry with the problem explained.
   If it still fails, the API returns **502**. A made-up movie never reaches the user.
5. **The movie in the reply comes from our database**, loaded by that id, not from Gemini's text.
6. **Prompt injection:** the user's text is wrapped in `<user_request>` tags and declared as data, not instructions.
   A fake closing tag inside the text is removed. (Tested with a fake Gemini server.)
7. If the key is missing the endpoint answers 503; the rest of the API is not affected.

## Project structure

```
cmd/api              starts the server and connects all parts together
cmd/seed             loads data/movies.json into the database
data/movies.json     the 20 starter movies
internal/config      reads .env / environment variables
internal/database    PostgreSQL connection + migrations/ (the table definitions, run at start)
internal/models      structs that match the tables (no logic)
internal/repository  ALL SQL queries (one file per table)
internal/auth        bcrypt passwords + JWT tokens
internal/middleware  RequireAuth (token check) and CORS
internal/handler     the endpoints + routes.go (list of all URLs)
internal/gemini      the grounded AI recommendation
```

A request flows downward and never back up:

```
HTTP request -> middleware (CORS, token) -> handler -> repository -> database -> PostgreSQL
                                               \-> auth (hash, token)   \-> gemini (HTTP to Google)
```
