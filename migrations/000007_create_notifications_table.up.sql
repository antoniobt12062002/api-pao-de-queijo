CREATE TABLE IF NOT EXISTS notifications (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    round_id UUID NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
    type     VARCHAR(30) NOT NULL CHECK (type IN ('round_announced', 'participation_open', 'round_closed', 'reminder')),
    sent_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    channel  VARCHAR(10) NOT NULL CHECK (channel IN ('push', 'web'))
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications (user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_round_id ON notifications (round_id);
