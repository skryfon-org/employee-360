-- Working days a request covers (tenant weekly off days excluded; half day = 0.5),
-- fixed at apply time so every listing agrees. Backfilled from each tenant's
-- current weekly_off_days (default {0,6} when the tenant has no settings row).
ALTER TABLE leave_requests ADD COLUMN days NUMERIC(6, 1) NOT NULL DEFAULT 0;

UPDATE leave_requests lr
SET days = CASE
    WHEN lr.day_part <> 'full' THEN
        CASE WHEN EXTRACT(DOW FROM lr.start_date)::int = ANY (off.d) THEN 0 ELSE 0.5 END
    ELSE (
        SELECT COUNT(*)
        FROM generate_series(lr.start_date::timestamp, lr.end_date::timestamp, INTERVAL '1 day') g
        WHERE NOT (EXTRACT(DOW FROM g)::int = ANY (off.d))
    )
END
FROM (
    SELECT t.id AS tenant_id,
           COALESCE(os.weekly_off_days, ARRAY[0, 6]::smallint[])::int[] AS d
    FROM tenants t
    LEFT JOIN organization_settings os ON os.tenant_id = t.id
) off
WHERE off.tenant_id = lr.tenant_id;
