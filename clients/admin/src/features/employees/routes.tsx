import { Navigate, Route } from 'react-router-dom'
import EmployeesPage from './pages/EmployeesPage'
import EmployeeDetailLayout from './pages/EmployeeDetailLayout'
import {
  AddressSection,
  AllocationsSection,
  AssignmentsSection,
  EmergencyContactsSection,
  EmploymentSection,
  LeaveSection,
  OverviewSection,
  PersonalSection,
  RoleSection,
  StatusLogsSection,
} from './pages/EmployeeSectionPages'
import { EMPLOYEE_SECTIONS } from './sections'

/** Authenticated routes; mount inside the RequireAuth + AdminShell layout. */
export const employeeRoutes = (
  <>
    <Route path="/employees" element={<EmployeesPage />} />
    <Route path="/employees/:id" element={<EmployeeDetailLayout />}>
      <Route index element={<Navigate to={EMPLOYEE_SECTIONS[0].path} replace />} />
      <Route path="overview" element={<OverviewSection />} />
      <Route path="employment" element={<EmploymentSection />} />
      <Route path="personal" element={<PersonalSection />} />
      <Route path="address" element={<AddressSection />} />
      <Route path="emergency-contacts" element={<EmergencyContactsSection />} />
      <Route path="assignments" element={<AssignmentsSection />} />
      <Route path="allocations" element={<AllocationsSection />} />
      <Route path="status-logs" element={<StatusLogsSection />} />
      <Route path="leave" element={<LeaveSection />} />
      <Route path="role" element={<RoleSection />} />
    </Route>
  </>
)
