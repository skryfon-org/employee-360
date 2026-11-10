-- Add description and is_active columns, and case-insensitive unique name per tenant for active (non-deleted) positions.
ALTER TABLE positions
    ADD COLUMN description TEXT,
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- Dedupe: rename case-insensitive duplicate active names per tenant so the unique
-- index can be created. The oldest row keeps its name; the others get " (2)", " (3)", ...
-- The base name is truncated so the result stays within 100 characters.
WITH ranked AS (
    SELECT id,
           name,
           ROW_NUMBER() OVER (
               PARTITION BY tenant_id, LOWER(name)
               ORDER BY created_at ASC, id ASC
           ) AS rn
    FROM positions
    WHERE deleted_at IS NULL
)
UPDATE positions p
SET name = LEFT(r.name, 100 - LENGTH(' (' || r.rn || ')')) || ' (' || r.rn || ')'
FROM ranked r
WHERE p.id = r.id
  AND r.rn > 1;

CREATE UNIQUE INDEX uq_positions_tenant_id_name ON positions (tenant_id, LOWER(name)) WHERE deleted_at IS NULL;
