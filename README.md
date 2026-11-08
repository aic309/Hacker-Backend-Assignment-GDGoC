# Movie Watchlist API

Backend assignment for **GDGoC UGM Hacker (Back-End)** by Muhammad Alaric Athaya Iskandar.

A REST API for a movie watchlist and review app. Users register and log in, browse 20 movies, keep a watchlist, log watched movies with a 1 to 5 rating and a review, and ask **Google Gemini** for a recommendation that can only be a movie from the database.

**Stack:** Go (standard library `net/http`), PostgreSQL (`pgx`), bcrypt, JWT, Google Gemini.

## Live demo

| | Link |
| --- | --- |
| Live frontend | https://hacker-backend-assignment-gdgoc-fro.vercel.app/ |
| Live API | https://hacker-backend-assignment-gdgoc.onrender.com/api |
| Demo video | https://drive.google.com/file/d/17JqMgSEB2FTQFxHUbGyE72rz7v_5Q7M3/view?usp=sharing |

The API runs on Render's free plan, so it sleeps when nobody uses it. The first request can take up to a minute. Open [`/api/health`](https://hacker-backend-assignment-gdgoc.onrender.com/api/health) first to wake it up.

Deployed with: **Neon** (PostgreSQL), **Render** (Go API), **Vercel** (frontend).

## What is in this repo

```
backend/    the Go API (code, tests, .env.example)
frontend/   the starter frontend, works with the API without changes
data/       the 20 starter movies
postman/    the Postman collection to try every endpoint
docs/       the ERD picture
```

More detail about the backend code is in [`backend/README.md`](backend/README.md).

## Database design (ERD)

![ERD](docs/erd_chen.png)

| Table | Columns |
| --- | --- |
| `users` | `id` PK, `username`, `email` **UNIQUE**, `password_hash`, `created_at` |
| `movies` | `id` PK, `title`, `year`, `director`, `genres`, `rating`, `duration`, `synopsis`, `poster_url`, `cast_members` |
| `watchlist` | `id` PK, `user_id` FK, `movie_id` FK, `created_at`, **UNIQUE(user_id, movie_id)** |
| `watched` | `id` PK, `user_id` FK, `movie_id` FK, `rating`, `review`, `created_at`, **UNIQUE(user_id, movie_id)**, **CHECK(rating BETWEEN 1 AND 5)** |

- `watchlist` and `watched` are link tables (many-to-many between users and movies). They are two tables because they mean different things: "I want to watch this" and "I watched this and rated it".
- The database enforces the rules, not only the code: `UNIQUE`, `CHECK`, and foreign keys with `ON DELETE CASCADE`. If my code has a bug, bad data still cannot get in.
- The tables are created by `backend/internal/database/migrations/001_init.sql`, which runs when the server starts.

## Quick start (run it on your computer)

**You need:** Go 1.23+, PostgreSQL 14+ (running), and a free Gemini API key from https://aistudio.google.com

**Step 1: Download the project and make the settings file**

```bash
git clone https://github.com/aic309/Hacker-Backend-Assignment-GDGoC.git
cd Hacker-Backend-Assignment-GDGoC/backend
createdb moviedb
cp .env.example .env
openssl rand -hex 32
```

`git clone` downloads the project. You can also click **Code → Download ZIP** on GitHub and unzip it. `createdb moviedb` makes an empty database. `openssl rand -hex 32` prints a random secret.

**Step 2: Fill in `.env` (do not skip this)**

The `.env` file holds the settings and secrets of the server. Step 1 made it as a copy of `.env.example`, and it still has placeholder values. You must change 3 lines.

*2a. Get a Gemini API key (free)*

1. Go to https://aistudio.google.com and sign in with a Google account.
2. Click **Get API key**, then **Create API key**.
3. Copy the key. It starts with `AIza`.

*2b. Open the file*

In the `backend` folder, run one of these:

```bash
open -e .env
```

```bash
nano .env
```

- `open -e .env` opens it in TextEdit (Mac). Save with `Cmd + S`.
- `nano .env` edits it in the terminal. Save with `Ctrl + O`, then `Enter`, then exit with `Ctrl + X`.
- On Windows or Linux, open `.env` in any text editor. It is a hidden file because its name starts with a dot.

*2c. Change these 3 lines*

The file looks like this. Change only the lines marked with an arrow:

```
DATABASE_URL=postgres://localhost:5432/moviedb?sslmode=disable    <- check this
JWT_SECRET=change-me                                              <- change this
JWT_EXPIRE_MINUTES=60
GEMINI_API_KEY=                                                   <- change this
GEMINI_MODEL=gemini-3.5-flash-lite
PORT=8000
CORS_ORIGINS=*
```

| Line | What to put | Example |
| --- | --- | --- |
| `DATABASE_URL` | Must end with the database name from Step 1 (`moviedb`). The default works if PostgreSQL uses your computer username with no password. If you set a password, write `postgres://USER:PASSWORD@localhost:5432/moviedb?sslmode=disable`. | `postgres://localhost:5432/moviedb?sslmode=disable` |
| `JWT_SECRET` | The long text printed by `openssl rand -hex 32` in Step 1. At least 32 characters. | `d8a7516bd9a1c0be...` |
| `GEMINI_API_KEY` | The key from 2a. No quotes, no spaces. | `AIzaSy...` |

Rules: no spaces around `=`, no quotes, one value per line.

If you skip this step, the server stops with `JWT_SECRET must be at least 32 characters`. If `/recommend` gives 503, the Gemini key is empty.

**Step 3: Load the movies and start the server**

```bash
go mod tidy
go run ./cmd/seed
go run ./cmd/api
```

`go run ./cmd/seed` loads the 20 movies (safe to run many times). `go run ./cmd/api` starts the server at http://localhost:8000 and creates the tables automatically.

Check it works: open http://localhost:8000/api/health in a browser.

To try the frontend, open a second terminal:

```bash
cd Hacker-Backend-Assignment-GDGoC/frontend
python3 -m http.server 3000
```

Open http://localhost:3000. The frontend points to the live API by default. To use your local API, click **Settings** (top right) and set **API Base URL** to `http://localhost:8000/api`.

## Environment variables

| Name | Required | Meaning |
| --- | --- | --- |
| `DATABASE_URL` | yes | `postgres://USER:PASSWORD@HOST:PORT/DBNAME?sslmode=disable` |
| `JWT_SECRET` | yes | Signs the tokens. At least 32 characters, or the server will not start. |
| `GEMINI_API_KEY` | for `/recommend` | Without it, `/recommend` answers 503 and the rest of the API still works. |
| `GEMINI_MODEL` | no | Default `gemini-3.5-flash-lite`. Change it if Google retires the model. |
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
| GET | `/health` | no | 200 |

Errors are always JSON like `{"error": "message"}`.

| Code | When |
| --- | --- |
| 400 | Bad input: invalid email, password shorter than 8 or longer than 72 bytes, rating outside 1 to 5, empty prompt, broken JSON |
| 401 | No, invalid or expired token; wrong email or password |
| 404 | Movie not found; movie not in your watchlist; unknown URL |
| 409 | Email already registered; movie already in your watchlist or watched list |
| 413 | Request body over 1 MB |
| 500 | Unexpected server error (details go to the log, not to the client) |
| 502 | Gemini failed, or kept answering with a movie that is not in the database |
| 503 | `GEMINI_API_KEY` is not set |

## Postman

The collection is in `postman/`. Import it into Postman, run Register, then Login, and the token is saved for the other requests. The Register request uses a fixed email, so if you run the collection twice, use a new email or reset the database.

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

## Security

- **Passwords:** hashed with bcrypt. Only the hash is stored, and it is never returned. Passwords longer than 72 bytes are rejected, because bcrypt ignores the rest.
- **JWT:** HS256 with an expiry (default 60 minutes). Only HS256 is accepted, and tokens without an expiry are refused. `JWT_SECRET` must be at least 32 characters.
- **Data isolation:** the user id always comes from the token, never from the request body, so users cannot see or change each other's data.
- **Login errors:** wrong password and unknown email give the same message, so nobody can find out which emails exist.
- **SQL injection:** every query uses placeholders (`$1`, `$2`).
- **Input validation:** email format, password length, rating 1 to 5, review at most 2000 characters, prompt at most 500 characters, body at most 1 MB.
- **Errors:** unexpected errors are logged on the server, and the client only sees "internal server error".
- **Timeouts:** the HTTP server has read, write and idle timeouts.

## Gemini grounding

The goal: the AI may only recommend movies that exist in **my** database.

1. **The catalog is in the prompt.** Every request loads all movies from PostgreSQL and lists them (id, title, year, genres, director, rating, short synopsis).
2. **System instruction:** recommend exactly one movie, only from the catalog, and answer with its id.
3. **Forced JSON answer** (`responseSchema`): `{"movie_id": <int>, "reasoning": "<text>"}`. No free text to parse.
4. **I verify the id.** `movie_id` must be one of the ids I sent. If not, Gemini gets one retry. If it still fails, the API returns 502, so a made-up movie never reaches the user.
5. **The movie in the reply comes from my database**, loaded by that id, not from Gemini's text.
6. **Prompt injection:** the user's text is wrapped in `<user_request>` tags and treated as data, not instructions. A fake closing tag inside the text is removed.
7. If the key is missing, the endpoint answers 503 and the rest of the API is not affected.

## Code structure

```
backend/cmd/api              starts the server and connects all parts together
backend/cmd/seed             loads data/movies.json into the database
backend/internal/config      reads .env / environment variables
backend/internal/database    PostgreSQL connection + migrations (table definitions)
backend/internal/models      data shapes that match the tables (no logic)
backend/internal/repository  ALL SQL queries (one file per table)
backend/internal/auth        bcrypt passwords + JWT tokens
backend/internal/middleware  RequireAuth (token check) and CORS
backend/internal/handler     the endpoints + routes.go (list of all URLs)
backend/internal/gemini      the grounded AI recommendation
```

A request goes down the layers and never back up:

```
HTTP request -> middleware (CORS, token) -> handler -> repository -> PostgreSQL
                                               \-> auth (hash, token)
                                               \-> gemini (HTTP to Google)
```

- There is no separate service layer. The app is small, so the logic sits in the handlers and helper packages. I would add one if the business rules grew.
- Gemini is behind a small interface, so the tests use a fake Gemini and need no real key.
