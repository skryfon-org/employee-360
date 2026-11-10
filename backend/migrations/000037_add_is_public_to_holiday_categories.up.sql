-- holiday_categories.is_public: holidays in this category are company days off
-- (public holidays) and are excluded from leave day counts. Other categories
-- (e.g. optional holidays) count as normal working days for leave.
ALTER TABLE holiday_categories
    ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT FALSE;

-- Backfill: existing categories named "Public ..." are public.
UPDATE holiday_categories
SET is_public = TRUE
WHERE LOWER(TRIM(name)) = 'public holiday' OR LOWER(TRIM(name)) LIKE 'public%';

-- Recompute leave_requests.days (see 000036) counting only weekly off days and
-- non-deleted holidays whose category (same tenant) is public. A half day on a
-- weekly off day or public holiday counts 0.
UPDATE leave_requests lr
SET days = CASE
    WHEN lr.day_part <> 'full' THEN
        CASE WHEN EXTRACT(DOW FROM lr.start_date)::int = ANY (off.d)
                  OR EXISTS (SELECT 1 FROM holidays h
                             JOIN holiday_categories hc ON hc.id = h.category_id AND hc.tenant_id = h.tenant_id
                             WHERE h.tenant_id = lr.tenant_id AND h.deleted_at IS NULL
                               AND hc.is_public AND h.date = lr.start_date)
             THEN 0 ELSE 0.5 END
    ELSE (
        SELECT COUNT(*)
        FROM generate_series(lr.start_date::timestamp, lr.end_date::timestamp, INTERVAL '1 day') g
        WHERE NOT (EXTRACT(DOW FROM g)::int = ANY (off.d))
          AND NOT EXISTS (SELECT 1 FROM holidays h
                          JOIN holiday_categories hc ON hc.id = h.category_id AND hc.tenant_id = h.tenant_id
                          WHERE h.tenant_id = lr.tenant_id AND h.deleted_at IS NULL
                            AND hc.is_public AND h.date = g::date)
    )
END
FROM (
    SELECT t.id AS tenant_id,
           COALESCE(os.weekly_off_days, ARRAY[0, 6]::smallint[])::int[] AS d
    FROM tenants t
    LEFT JOIN organization_settings os ON os.tenant_id = t.id
) off
WHERE off.tenant_id = lr.tenant_id;
