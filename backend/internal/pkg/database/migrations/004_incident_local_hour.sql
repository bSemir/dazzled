-- Add an optional reporter-selected local hour for aggregated chart data.
ALTER TABLE incidents
    ADD COLUMN occurred_local_hour INTEGER
    CHECK (occurred_local_hour IS NULL OR occurred_local_hour BETWEEN 0 AND 23);
