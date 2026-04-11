CREATE TABLE IF NOT EXISTS rounds (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    date        DATE        NOT NULL UNIQUE,
    payer_id    UUID        NOT NULL REFERENCES users(id),
    status      TEXT        NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'open', 'closed', 'cancelled')),
    notify_at   TIMESTAMPTZ NOT NULL,
    closes_at   TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
