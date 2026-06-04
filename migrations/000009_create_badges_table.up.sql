CREATE TABLE IF NOT EXISTS badges (
    id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type      VARCHAR(20) NOT NULL CHECK (type IN ('queijeiro_fiel', 'papai_noel', 'nunca_foge', 'novo_na_fila', 'big_spender')),
    period    VARCHAR(7),
    earned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_badges_user_id ON badges (user_id);

-- Permanent badges (period = ''): unique per (user_id, type)
CREATE UNIQUE INDEX IF NOT EXISTS idx_badges_permanent
    ON badges (user_id, type) WHERE period = '';

-- Monthly badges (period != ''): unique per (user_id, type, period)
CREATE UNIQUE INDEX IF NOT EXISTS idx_badges_monthly
    ON badges (user_id, type, period) WHERE period != '';
