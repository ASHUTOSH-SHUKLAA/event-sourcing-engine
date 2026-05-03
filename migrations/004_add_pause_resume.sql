-- Add pause/resume support to subscriptions
ALTER TABLE subscriptions
  ADD COLUMN IF NOT EXISTS paused_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
  ADD COLUMN IF NOT EXISTS pause_days_remaining INTEGER DEFAULT NULL;

-- status can now be: 'active', 'paused', 'cancelled'
-- No enum change needed, it is stored as TEXT
