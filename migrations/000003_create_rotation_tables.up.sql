CREATE TABLE IF NOT EXISTS rotations (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    current_pos INT         NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS rotation_members (
    rotation_id UUID NOT NULL REFERENCES rotations(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id)     ON DELETE CASCADE,
    position    INT  NOT NULL,
    UNIQUE (rotation_id, user_id),
    UNIQUE (rotation_id, position)
);
