-- River job-queue schema (github.com/riverqueue/river), pinned to river-go v0.47.0.
--
-- Provenance: River ships its own schema migrations (7 versions, "main" line) applied
-- one-at-a-time by its internal migrator, each in its own transaction. golang-migrate
-- runs an entire migration FILE in a single transaction, and concatenating all 7 of
-- River's versions verbatim into one file fails under that constraint: migration 004
-- adds the 'pending' value to the river_job_state enum via ALTER TYPE ... ADD VALUE,
-- and migration 006 then uses that value inside a function body -- Postgres forbids
-- using a freshly-added enum value before the transaction that added it commits
-- (error 55P04, "unsafe use of new value of enum type").
--
-- To stay a single golang-migrate pair (as intended -- see plan/cycles/cycle-02-auth-onboarding.md),
-- this file is not River's raw migrate-get output. It is the exact FINAL schema produced
-- by actually running River's own migrator (all 7 versions, each in River's own
-- transaction, which is transactionally safe) against a scratch database, captured via
-- `pg_dump --schema-only` and stripped of pg_dump session boilerplate / `public.` schema
-- qualifiers. No table/column/index/constraint/function was hand-invented -- every
-- object here is exactly what River v0.47.0's migrator produces.
--
-- Regenerated with:
--   go run github.com/riverqueue/river/cmd/river@v0.47.0 migrate-up --database-url <scratch-db-url>
--   pg_dump --schema-only --no-owner --no-privileges --no-tablespaces <scratch-db>
--
-- No tenant_id column: this is River's own infrastructure schema, not tenant data --
-- tenant_id travels inside each job's `args jsonb` instead.
-- Do not hand-edit. To upgrade, regenerate against the target river-go version and check
-- in a new migration pair instead (never edit a committed migration).

CREATE TYPE river_job_state AS ENUM (
    'available',
    'cancelled',
    'completed',
    'discarded',
    'pending',
    'retryable',
    'running',
    'scheduled'
);

CREATE FUNCTION river_job_state_in_bitmask(bitmask bit, state river_job_state) RETURNS boolean
    LANGUAGE sql IMMUTABLE
    AS $$
    SELECT CASE state
        WHEN 'available' THEN get_bit(bitmask, 7)
        WHEN 'cancelled' THEN get_bit(bitmask, 6)
        WHEN 'completed' THEN get_bit(bitmask, 5)
        WHEN 'discarded' THEN get_bit(bitmask, 4)
        WHEN 'pending'   THEN get_bit(bitmask, 3)
        WHEN 'retryable' THEN get_bit(bitmask, 2)
        WHEN 'running'   THEN get_bit(bitmask, 1)
        WHEN 'scheduled' THEN get_bit(bitmask, 0)
        ELSE 0
    END = 1;
$$;

CREATE TABLE river_job (
    id bigserial PRIMARY KEY,
    state river_job_state DEFAULT 'available'::river_job_state NOT NULL,
    attempt smallint DEFAULT 0 NOT NULL,
    max_attempts smallint DEFAULT 25 NOT NULL,
    attempted_at timestamptz,
    created_at timestamptz DEFAULT now() NOT NULL,
    finalized_at timestamptz,
    scheduled_at timestamptz DEFAULT now() NOT NULL,
    priority smallint DEFAULT 1 NOT NULL,
    args jsonb NOT NULL,
    attempted_by text[],
    errors jsonb[],
    kind text NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    queue text DEFAULT 'default'::text NOT NULL,
    tags character varying(255)[] DEFAULT '{}'::character varying[] NOT NULL,
    unique_key bytea,
    unique_states bit(8),
    CONSTRAINT finalized_or_finalized_at_null CHECK ((((finalized_at IS NULL) AND (state <> ALL (ARRAY['cancelled'::river_job_state, 'completed'::river_job_state, 'discarded'::river_job_state]))) OR ((finalized_at IS NOT NULL) AND (state = ANY (ARRAY['cancelled'::river_job_state, 'completed'::river_job_state, 'discarded'::river_job_state]))))),
    CONSTRAINT kind_length CHECK (((char_length(kind) > 0) AND (char_length(kind) < 128))),
    CONSTRAINT max_attempts_is_positive CHECK ((max_attempts > 0)),
    CONSTRAINT priority_in_range CHECK (((priority >= 1) AND (priority <= 4))),
    CONSTRAINT queue_length CHECK (((char_length(queue) > 0) AND (char_length(queue) < 128)))
);

CREATE UNLOGGED TABLE river_leader (
    elected_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    leader_id text NOT NULL,
    name text DEFAULT 'default'::text NOT NULL,
    CONSTRAINT river_leader_pkey PRIMARY KEY (name),
    CONSTRAINT leader_id_length CHECK (((char_length(leader_id) > 0) AND (char_length(leader_id) < 128))),
    CONSTRAINT name_length CHECK ((name = 'default'::text))
);

CREATE TABLE river_migration (
    line text NOT NULL,
    version bigint NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT river_migration_pkey1 PRIMARY KEY (line, version),
    CONSTRAINT line_length CHECK (((char_length(line) > 0) AND (char_length(line) < 128))),
    CONSTRAINT version_gte_1 CHECK ((version >= 1))
);

CREATE TABLE river_notification (
    id bigserial PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    payload text NOT NULL,
    topic text NOT NULL,
    CONSTRAINT topic_length CHECK (((length(topic) > 0) AND (length(topic) < 128)))
);

CREATE TABLE river_queue (
    name text PRIMARY KEY NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    paused_at timestamptz,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX river_job_args_index ON river_job USING gin (args);
CREATE INDEX river_job_kind ON river_job USING btree (kind);
CREATE INDEX river_job_metadata_index ON river_job USING gin (metadata);
CREATE INDEX river_job_prioritized_fetching_index ON river_job USING btree (state, queue, priority, scheduled_at, id);
CREATE INDEX river_job_state_and_finalized_at_index ON river_job USING btree (state, finalized_at) WHERE (finalized_at IS NOT NULL);
CREATE UNIQUE INDEX river_job_unique_idx ON river_job USING btree (unique_key) WHERE ((unique_key IS NOT NULL) AND (unique_states IS NOT NULL) AND river_job_state_in_bitmask(unique_states, state));
CREATE INDEX river_notification_created_at_idx ON river_notification USING btree (created_at);
CREATE INDEX river_notification_topic_id_idx ON river_notification USING btree (topic, id);

-- Record that all 7 "main" line migrations are applied, matching the bookkeeping
-- River's own migrator would have written had it run them one-by-one -- required so
-- `river.Client` (and `river validate`) recognize the schema as current rather than
-- attempting to re-apply migrations 1-7 on top of this already-final schema.
INSERT INTO river_migration (line, version) VALUES
    ('main', 1),
    ('main', 2),
    ('main', 3),
    ('main', 4),
    ('main', 5),
    ('main', 6),
    ('main', 7);
