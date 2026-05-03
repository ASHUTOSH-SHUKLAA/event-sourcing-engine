-- Upgrade subscriptions table with professional billing fields
ALTER TABLE subscriptions 
ADD COLUMN current_period_end TIMESTAMP WITH TIME ZONE DEFAULT (now() + interval '30 days'),
ADD COLUMN cancel_at_period_end BOOLEAN DEFAULT FALSE,
ADD COLUMN total_revenue_cents INTEGER DEFAULT 0;

-- Update existing records to have a valid period end
UPDATE subscriptions SET current_period_end = (now() + interval '30 days') WHERE current_period_end IS NULL;
