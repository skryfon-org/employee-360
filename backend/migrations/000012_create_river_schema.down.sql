-- Reverse of 000012_create_river_schema.up.sql. See that file for provenance notes.

DROP INDEX IF EXISTS river_notification_topic_id_idx;
DROP INDEX IF EXISTS river_notification_created_at_idx;
DROP INDEX IF EXISTS river_job_unique_idx;
DROP INDEX IF EXISTS river_job_state_and_finalized_at_index;
DROP INDEX IF EXISTS river_job_prioritized_fetching_index;
DROP INDEX IF EXISTS river_job_metadata_index;
DROP INDEX IF EXISTS river_job_kind;
DROP INDEX IF EXISTS river_job_args_index;

DROP TABLE IF EXISTS river_queue;
DROP TABLE IF EXISTS river_notification;
DROP TABLE IF EXISTS river_migration;
DROP TABLE IF EXISTS river_leader;
DROP TABLE IF EXISTS river_job;

DROP FUNCTION IF EXISTS river_job_state_in_bitmask(bit, river_job_state);

DROP TYPE IF EXISTS river_job_state;
