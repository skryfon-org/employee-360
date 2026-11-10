-- Revert adding description column and unique index on (tenant_id, LOWER(name)) to departments.
DROP INDEX IF EXISTS uq_departments_tenant_id_name;

ALTER TABLE departments
    DROP COLUMN IF EXISTS description;
