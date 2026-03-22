-- 002_add_items.sql — Add items table for E2E tests

CREATE TABLE IF NOT EXISTS items (
    id         TEXT PRIMARY KEY,
    order_id   TEXT NOT NULL,
    sku        TEXT NOT NULL,
    quantity   INTEGER NOT NULL DEFAULT 1,
    price      NUMERIC(10, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
