CREATE TABLE IF NOT EXISTS participations (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    round_id     UUID        NOT NULL REFERENCES rounds(id),
    user_id      UUID        NOT NULL REFERENCES users(id),
    quantity     INT         NOT NULL CHECK (quantity >= 1),
    confirmed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (round_id, user_id)
);
