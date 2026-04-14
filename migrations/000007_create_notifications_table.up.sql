CREATE TABLE notifications (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    round_id UUID NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
    type     VARCHAR(30) NOT NULL CHECK (type IN ('round_announced', 'participation_open', 'round_closed', 'reminder')),
    sent_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    channel  VARCHAR(10) NOT NULL CHECK (channel IN ('push', 'web'))
);
