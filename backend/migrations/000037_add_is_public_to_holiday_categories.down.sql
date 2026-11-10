-- Restore the 000036 count (every non-deleted holiday excluded), then drop the flag.
UPDATE leave_requests lr
SET days = CASE
    WHEN lr.day_part <> 'full' THEN
        CASE WHEN EXTRACT(DOW FROM lr.start_date)::int = ANY (off.d)
                  OR EXISTS (SELECT 1 FROM holidays h
                             WHERE h.tenant_id = lr.tenant_id AND h.deleted_at IS NULL AND h.date = lr.start_date)
             THEN 0 ELSE 0.5 END
    ELSE (
        SELECT COUNT(*)
        FROM generate_series(lr.start_date::timestamp, lr.end_date::timestamp, INTERVAL '1 day') g
        WHERE NOT (EXTRACT(DOW FROM g)::int = ANY (off.d))
          AND NOT EXISTS (SELECT 1 FROM holidays h
                          WHERE h.tenant_id = lr.tenant_id AND h.deleted_at IS NULL AND h.date = g::date)
    )
END
FROM (
    SELECT t.id AS tenant_id,
           COALESCE(os.weekly_off_days, ARRAY[0, 6]::smallint[])::int[] AS d
    FROM tenants t
    LEFT JOIN organization_settings os ON os.tenant_id = t.id
) off
WHERE off.tenant_id = lr.tenant_id;

ALTER TABLE holiday_categories DROP COLUMN IF EXISTS is_public;
