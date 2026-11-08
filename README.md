# Movie Watchlist API

Backend assignment for **GDGoC UGM Hacker (Back-End)** by Muhammad Alaric Athaya Iskandar.

A REST API for a movie watchlist and review app. Users register and log in, browse 20 movies, keep a watchlist, log watched movies with a 1 to 5 rating and a review, and ask **Google Gemini** for a recommendation that can only be a movie from the database.

**Stack:** Go (standard library `net/http`), PostgreSQL (`pgx`), bcrypt, JWT, Google Gemini.



## What is in this repo

```
backend/    the Go API (code, tests, .env.example)
frontend/   the starter frontend, works with the API without changes
data/       the 20 starter movies
postman/    the Postman collection to try every endpoint
docs/       the ERD and Postman screenshots
```

The full API reference, database design, security notes and Gemini explanation are in [`backend/README.md`](backend/README.md).

## Quick start

**You need:** Go 1.23+, PostgreSQL 14+ (running), and a free Gemini API key from https://aistudio.google.com

```bash
cd backend
createdb moviedb
cp .env.example .env
openssl rand -hex 32
go mod tidy
go run ./cmd/seed
go run ./cmd/api
```

1. `createdb moviedb` makes an empty database.
2. Open `.env` and fill in the values from the table below. Use the output of `openssl rand -hex 32` as `JWT_SECRET`.
3. `go run ./cmd/seed` loads the 20 movies. It is safe to run many times.
4. `go run ./cmd/api` starts the server at http://localhost:8000. The tables are created automatically.

To try the frontend, open a second terminal:

```bash
cd frontend
python3 -m http.server 3000
```

Then open http://localhost:3000.

## Environment variables

| Name | Required | Meaning |
| --- | --- | --- |
| `DATABASE_URL` | yes | `postgres://USER:PASSWORD@HOST:PORT/DBNAME?sslmode=disable` |
| `JWT_SECRET` | yes | Signs the tokens. At least 32 characters, or the server will not start. |
| `GEMINI_API_KEY` | for `/recommend` | Without it, `/recommend` answers 503 and the rest of the API still works. |
| `GEMINI_MODEL` | no | I tested with `gemini-3.5-flash-lite`. Set it if the default model is not available for your key. |
| `JWT_EXPIRE_MINUTES` | no | Default `60` |
| `PORT` | no | Default `8000` |
| `CORS_ORIGINS` | no | Allowed frontend addresses, comma separated. Default `*` |

Never commit your `.env`. It is in `.gitignore`. Use `backend/.env.example` as the template.

## Endpoints

Base URL `http://localhost:8000/api`. Protected routes need `Authorization: Bearer <token>`.

| Method | URL | Auth | Success |
| --- | --- | --- | --- |
| POST | `/auth/register` | no | 201 |
| POST | `/auth/login` | no | 200 with token |
| GET | `/movies` | no | 200 (filters: `genre`, `search`, `shuffle`) |
| GET | `/movies/{id}` | no | 200 |
| POST | `/movies/recommend` | no | 200 with `movie_id`, `reasoning`, `movie` |
| GET | `/watchlist` | yes | 200 |
| POST | `/watchlist` | yes | 201 |
| DELETE | `/watchlist/{movieId}` | yes | 204 |
| GET | `/watched` | yes | 200 |
| POST | `/watched` | yes | 201 (rating 1 to 5) |

Errors are always JSON like `{"error": "message"}`. Codes used: 400, 401, 404, 409, 413, 500, 502, 503. The full list of when each one is used is in `backend/README.md`.

A Postman collection is in `postman/`. Import it, register, log in, and the token is saved for the other requests.

## Tests

```bash
cd backend
go test ./...
```

Unit tests (JWT, bcrypt, middleware, Gemini grounding with a fake Gemini server) always run. The API tests need an empty test database, and their tables are dropped at the start of each test, so never use your real database:

```bash
createdb moviedb_test
TEST_DATABASE_URL="postgres://localhost:5432/moviedb_test?sslmode=disable" go test ./...
```

No API key or internet is needed for the tests.

## Design decisions

**Database**
- 4 tables: `users`, `movies`, `watchlist`, `watched`. The last two are join tables (many-to-many between users and movies).
- Watchlist and watched are separate because they mean different things: "I want to watch this" and "I watched this and rated it". They also have different columns.
- The database enforces the rules, not only the code: `UNIQUE` email, `UNIQUE(user_id, movie_id)`, `CHECK(rating BETWEEN 1 AND 5)`, foreign keys with `ON DELETE CASCADE`. If my code has a bug, bad data still cannot get in.

**Security**
- Passwords are hashed with bcrypt. Only the hash is stored.
- JWT with HS256 and an expiry. Only HS256 is accepted, and tokens without an expiry are refused.
- The user id always comes from the token, never from the request body, so users cannot see each other's data.
- Wrong password and unknown email give the same error, so nobody can find out which emails exist.
- All SQL uses placeholders, so SQL injection does not work. Request bodies have a size limit and the server has timeouts.

**Gemini grounding**
- All movies from the database are put in the prompt, and Gemini is told to choose only from that list.
- The answer is forced into JSON with `movie_id` and `reasoning`.
- I check that `movie_id` is really in the catalog. If not, I retry once, then return 502.
- The movie in the response is loaded from my database by id, not taken from Gemini's text.
- The user's text is wrapped in `<user_request>` tags and treated as data, so it cannot change the instructions.

**Code structure**
- Layers with one job each: handler (HTTP) → repository (all SQL) → PostgreSQL. Models are shared data shapes. Auth, Gemini and config are separate packages.
- There is no separate service layer. The app is small, so the logic sits in the handlers and helper packages. I would add one if the business rules grew.
- Gemini is behind a small interface, so tests use a fake Gemini and need no real key.
