-- Revert adding the is_active column to departments.
ALTER TABLE departments
    DROP COLUMN IF EXISTS is_active;
