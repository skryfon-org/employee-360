-- organization_settings holds only derived/default settings (rows are backfilled or
-- lazily created with defaults), so dropping the table is safe. Verified up->down->up.
DROP TABLE IF EXISTS organization_settings;
