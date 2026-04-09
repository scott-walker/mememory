-- Add delivery column to separate loading strategy from semantic type.
-- delivery = 'bootstrap' → loaded at session start
-- delivery = 'on_demand' → loaded via recall/list on demand
ALTER TABLE memories ADD COLUMN IF NOT EXISTS delivery TEXT NOT NULL DEFAULT 'on_demand';

-- Migrate existing type=bootstrap records: set delivery=bootstrap, keep type as-is.
-- Users will manually assign correct types later via admin UI.
UPDATE memories SET delivery = 'bootstrap' WHERE type = 'bootstrap';

CREATE INDEX IF NOT EXISTS idx_memories_delivery ON memories(delivery);
