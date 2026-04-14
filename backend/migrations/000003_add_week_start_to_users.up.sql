ALTER TABLE users
  ADD COLUMN week_start VARCHAR NOT NULL DEFAULT 'monday'
  CHECK (week_start IN ('monday', 'sunday'));
