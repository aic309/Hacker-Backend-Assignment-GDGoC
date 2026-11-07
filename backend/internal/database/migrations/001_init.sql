-- Tables for the movie watchlist API (PostgreSQL).
-- Runs on every start, so every statement uses IF NOT EXISTS (safe to run again).

-- Same columns as data/seed.sql in the starter pack, so the frontend gets the
-- field names it expects.
CREATE TABLE IF NOT EXISTS movies (
    id           SERIAL PRIMARY KEY,
    title        VARCHAR(255)  NOT NULL,
    year         INT           NOT NULL,
    director     VARCHAR(255)  NOT NULL,
    genres       TEXT          NOT NULL,
    rating       NUMERIC(3, 1) NOT NULL,
    duration     VARCHAR(50)   NOT NULL,
    synopsis     TEXT          NOT NULL,
    poster_url   TEXT          NOT NULL,
    cast_members TEXT          NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id            SERIAL PRIMARY KEY,
    -- Display name shown in the app. Not unique: the email is the real identity.
    username      VARCHAR(50)  NOT NULL,
    -- UNIQUE also creates an index, so login (find by email) is fast.
    email         VARCHAR(255) NOT NULL UNIQUE,
    -- Only the bcrypt hash is stored, never the real password.
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Movies a user WANTS to watch. One row = one user + one movie.
CREATE TABLE IF NOT EXISTS watchlist (
    id       SERIAL PRIMARY KEY,
    -- ON DELETE CASCADE: delete the user (or movie) and these rows go too.
    user_id  INT         NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    movie_id INT         NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- The database itself blocks the same movie twice for one user (API answers 409).
    CONSTRAINT uq_watchlist_user_movie UNIQUE (user_id, movie_id)
);
-- The UNIQUE above is an index that starts with user_id. This one makes
-- "which users have this movie" and cascade deletes fast.
CREATE INDEX IF NOT EXISTS idx_watchlist_movie_id ON watchlist(movie_id);

-- Movies a user HAS watched, with a 1-5 rating and a written review.
CREATE TABLE IF NOT EXISTS watched (
    id         SERIAL PRIMARY KEY,
    user_id    INT         NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    movie_id   INT         NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    rating     INT         NOT NULL,
    review     TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_watched_user_movie UNIQUE (user_id, movie_id),
    -- The database refuses a rating outside 1-5, even if the code has a bug.
    CONSTRAINT ck_watched_rating_1_5 CHECK (rating BETWEEN 1 AND 5)
);
CREATE INDEX IF NOT EXISTS idx_watched_movie_id ON watched(movie_id);
