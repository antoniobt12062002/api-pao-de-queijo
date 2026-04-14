CREATE TABLE device_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT NOT NULL UNIQUE,
    platform   VARCHAR(10) NOT NULL CHECK (platform IN ('web', 'android', 'ios')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
