import type { EmployeeDetail } from '../queries/employeeQueries'
import type { EmploymentValues, PersonalAddressValues } from './employeeSchemas'

export function employmentDefaults(e: EmployeeDetail): EmploymentValues {
  const p = e.profile ?? {}
  return {
    employee_code: e.employee_code ?? p.employee_code ?? '',
    employment_type: e.employment_type ?? p.employment_type ?? '',
    employment_status: (e.employment_status ?? p.employment_status ?? 'active') as EmploymentValues['employment_status'],
    joining_date: e.joining_date ?? p.joining_date ?? '',
    probation_end_date: p.probation_end_date ?? '',
    confirmation_date: p.confirmation_date ?? '',
    exit_date: p.exit_date ?? '',
    work_location: e.work_location ?? p.work_location ?? '',
    department_id: e.department_id ?? '',
    position_id: e.position_id ?? '',
    manager_id: e.manager_id ?? p.manager_id ?? '',
    effective_date: '',
    change_reason: '',
  }
}

export function personalDefaults(e: EmployeeDetail): PersonalAddressValues {
  const p = e.profile ?? {}
  return {
    first_name: e.first_name ?? '',
    last_name: e.last_name ?? '',
    gender: p.gender ?? '',
    date_of_birth: p.date_of_birth ?? '',
    blood_group: p.blood_group ?? '',
    marital_status: p.marital_status ?? '',
    nationality: p.nationality ?? '',
    phone: p.phone ?? '',
    personal_email: p.personal_email ?? '',
    address_line1: p.address_line1 ?? '',
    address_line2: p.address_line2 ?? '',
    city: p.city ?? '',
    state: p.state ?? '',
    country: p.country ?? '',
    pincode: p.pincode ?? '',
  }
}
