CREATE TABLE IF NOT EXISTS configs (
    key   VARCHAR(100) PRIMARY KEY,
    value TEXT         NOT NULL
);

INSERT INTO configs (key, value) VALUES
    ('notify_at',             '08:00'),
    ('round_window_minutes',  '30'),
    ('price_per_unit',        '2.50')
ON CONFLICT (key) DO NOTHING;
