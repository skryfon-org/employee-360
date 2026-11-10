DROP INDEX IF EXISTS uq_emergency_contacts_primary;
DROP INDEX IF EXISTS idx_emergency_contacts_user_id;
DROP INDEX IF EXISTS idx_emergency_contacts_tenant_id;
DROP TABLE IF EXISTS emergency_contacts;

DROP INDEX IF EXISTS uq_employee_profiles_tenant_code;
DROP INDEX IF EXISTS uq_employee_profiles_tenant_user;
DROP INDEX IF EXISTS idx_employee_profiles_manager_id;
DROP INDEX IF EXISTS idx_employee_profiles_user_id;
DROP INDEX IF EXISTS idx_employee_profiles_tenant_id;
DROP TABLE IF EXISTS employee_profiles;
