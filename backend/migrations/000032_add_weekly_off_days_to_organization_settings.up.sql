-- Per-tenant weekly off days (0 = Sunday .. 6 = Saturday), used by the monthly
-- status report to compute working days. Existing rows (and tenants without a row,
-- handled by the application) get the default {0,6} (Sunday, Saturday).
-- Uniqueness is enforced by the usecase; the CHECK bounds the values and forbids
-- an empty or all-seven-days list. Tenant scoping is inherited from the row.
ALTER TABLE organization_settings
    ADD COLUMN weekly_off_days SMALLINT[] NOT NULL DEFAULT '{0,6}',
    ADD CONSTRAINT chk_organization_settings_weekly_off_days
        CHECK (cardinality(weekly_off_days) BETWEEN 1 AND 6
               AND weekly_off_days <@ ARRAY[0,1,2,3,4,5,6]::smallint[]);
