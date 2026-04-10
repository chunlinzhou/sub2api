ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS prompt_policy JSONB NOT NULL DEFAULT '{}'::jsonb;
