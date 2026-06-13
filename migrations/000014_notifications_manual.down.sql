ALTER TABLE notifications ALTER COLUMN round_id SET NOT NULL;
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_type_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_type_check
  CHECK (type IN ('round_announced','participation_open','round_closed','reminder'));
