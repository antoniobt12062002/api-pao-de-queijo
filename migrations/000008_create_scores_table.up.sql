CREATE TABLE IF NOT EXISTS scores (
    user_id              UUID        PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    times_paid           INT         NOT NULL DEFAULT 0,
    times_participated   INT         NOT NULL DEFAULT 0,
    total_amount_spent   NUMERIC     NOT NULL DEFAULT 0,
    skip_count           INT         NOT NULL DEFAULT 0,
    current_streak       INT         NOT NULL DEFAULT 0,
    score                NUMERIC     NOT NULL DEFAULT 0,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
