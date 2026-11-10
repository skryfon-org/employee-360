-- Recompute leave_requests.days so tenant holidays are non-working like weekly
-- off days (000035 excluded only weekly off days). Uses each tenant's current
-- weekly_off_days (default {0,6}) and non-deleted holidays. Half days on a
-- weekly off day or holiday count 0.
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
