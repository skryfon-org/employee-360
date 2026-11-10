-- Employee profiles (1:1 with users) and emergency contacts (many per user).
-- users stays identity/auth only; personal, employment and address details live
-- here. All rows are tenant-scoped. Soft-delete + audit columns are inline;
-- actor columns are plain UUIDs with NO foreign key (per 000015).
--
-- profile_completed_at: set by the application when the employee's mandatory
-- self-filled fields (and at least one emergency contact) are present; cleared if
-- they stop being present. Drives the 'profile_incomplete' gating.
CREATE TABLE employee_profiles (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    user_id              UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,

    -- Personal
    employee_code        TEXT,
    gender               TEXT,
    date_of_birth        DATE,
    blood_group          TEXT,
    marital_status       TEXT,
    nationality          TEXT,
    phone                TEXT,
    personal_email       TEXT,

    -- Employment
    joining_date         DATE,
    employment_type      TEXT,
    employment_status    TEXT NOT NULL DEFAULT 'active',
    probation_end_date   DATE,
    confirmation_date    DATE,
    exit_date            DATE,
    work_location        TEXT,
    manager_id           UUID REFERENCES users (id) ON DELETE SET NULL,

    -- Address
    address_line1        TEXT,
    address_line2        TEXT,
    city                 TEXT,
    state                TEXT,
    country              TEXT,
    pincode              TEXT,

    -- Onboarding state
    profile_completed_at TIMESTAMPTZ,

    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by           UUID,
    updated_by           UUID,
    deleted_by           UUID,
    deleted_at           TIMESTAMPTZ,

    CONSTRAINT chk_employee_profiles_gender CHECK (gender IS NULL OR gender IN ('male', 'female', 'other', 'prefer_not_to_say')),
    CONSTRAINT chk_employee_profiles_blood_group CHECK (blood_group IS NULL OR blood_group IN ('A+', 'A-', 'B+', 'B-', 'AB+', 'AB-', 'O+', 'O-')),
    CONSTRAINT chk_employee_profiles_marital_status CHECK (marital_status IS NULL OR marital_status IN ('single', 'married', 'divorced', 'widowed', 'other')),
    CONSTRAINT chk_employee_profiles_employment_type CHECK (employment_type IS NULL OR employment_type IN ('full_time', 'part_time', 'contract', 'intern', 'consultant')),
    CONSTRAINT chk_employee_profiles_employment_status CHECK (employment_status IN ('active', 'probation', 'on_leave', 'notice_period', 'resigned', 'terminated')),
    CONSTRAINT chk_employee_profiles_exit_after_joining CHECK (exit_date IS NULL OR joining_date IS NULL OR exit_date >= joining_date)
);

CREATE INDEX idx_employee_profiles_tenant_id ON employee_profiles (tenant_id);
CREATE INDEX idx_employee_profiles_user_id ON employee_profiles (user_id);
CREATE INDEX idx_employee_profiles_manager_id ON employee_profiles (manager_id);
CREATE UNIQUE INDEX uq_employee_profiles_tenant_user ON employee_profiles (tenant_id, user_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_employee_profiles_tenant_code ON employee_profiles (tenant_id, employee_code) WHERE deleted_at IS NULL AND employee_code IS NOT NULL;

CREATE TABLE emergency_contacts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    relationship    TEXT NOT NULL,
    phone           TEXT NOT NULL,
    alternate_phone TEXT,
    email           TEXT,
    is_primary      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID,
    updated_by      UUID,
    deleted_by      UUID,
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_emergency_contacts_tenant_id ON emergency_contacts (tenant_id);
CREATE INDEX idx_emergency_contacts_user_id ON emergency_contacts (user_id);
-- At most one primary contact per user among live rows.
CREATE UNIQUE INDEX uq_emergency_contacts_primary ON emergency_contacts (tenant_id, user_id) WHERE is_primary AND deleted_at IS NULL;
