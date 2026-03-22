-- 001_init.sql — Create test tables for E2E tests

CREATE TABLE IF NOT EXISTS posts (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    content    TEXT,
    author     TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orders (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    total      NUMERIC(10, 2) NOT NULL DEFAULT 0,
    status     TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS products (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    price         NUMERIC(10, 2) NOT NULL DEFAULT 0,
    photo__file   TEXT,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);
