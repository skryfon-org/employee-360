import { Link, useNavigate, useParams } from 'react-router-dom'
import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { getErrorCode, getErrorMessage } from '@employee360/api-client'
import { Card, FormField, InlineAlert, PageContainer, PageHeader, SubmitButton } from '@employee360/ui'
import {
  PROJECT_STATUSES,
  PROJECT_STATUS_LABELS,
  projectFormSchema,
  type ProjectFormValues,
} from '../schemas/projectSchemas'
import { useDebouncedValue } from '../../../hooks/useDebouncedValue'
import { useCreateProjectMutation, useProjectCodeSuggestionQuery, useProjectQuery, useUpdateProjectMutation, type Project } from '../queries/projectQueries'
import { ProjectStatusBadge } from '../components/ProjectStatusBadge'
import { SelectField, secondaryButtonCls } from '../../employees/components/fields'
import { useToast } from '../../../hooks/useToast'

function ProjectForm({ project }: { project?: Project }) {
  const toast = useToast()
  const create = useCreateProjectMutation()
  const update = useUpdateProjectMutation()
  const editing = Boolean(project)
  const pending = create.isPending || update.isPending
  const navigate = useNavigate()
  const backTo = project?.id ? `/projects/${project.id}` : '/projects'

  const {
    register,
    handleSubmit,
    setError,
    setFocus,
    control,
    formState: { errors },
  } = useForm<ProjectFormValues>({
    resolver: zodResolver(projectFormSchema),
    defaultValues: {
      name: project?.name ?? '',
      code: project?.code ?? '',
      description: project?.description ?? '',
      client_name: project?.client_name ?? '',
      status: (project?.status as ProjectFormValues['status']) ?? 'active',
      start_date: project?.start_date ?? '',
      end_date: project?.end_date ?? '',
    },
  })

  const onSubmit = (v: ProjectFormValues) => {
    if (editing && !v.code) {
      setError('code', { type: 'validate', message: 'Code is required' })
      setFocus('code')
      return
    }
    const body = {
      name: v.name,
      // Blank on create = let the server generate it from the name.
      code: v.code || undefined,
      description: v.description,
      client_name: v.client_name || undefined,
      status: v.status,
      start_date: v.start_date,
      end_date: v.end_date || undefined,
    }
    const handlers = {
      onSuccess: (saved: Project) => {
        toast.success(editing ? `Project "${v.name}" updated.` : `Project "${v.name}" created.`)
        navigate(editing ? backTo : saved.id ? `/projects/${saved.id}` : '/projects')
      },
      onError: (err: unknown) => {
        // The backend returns CONFLICT for a duplicate project code within the tenant.
        if (getErrorCode(err) === 'CONFLICT') {
          setError('code', { type: 'server', message: 'A project with this code already exists.' })
          setFocus('code')
          return
        }
        toast.error(getErrorMessage(err, editing ? 'Could not update project.' : 'Could not create project.'))
      },
    }
    if (project?.id) update.mutate({ id: project.id, body }, handlers)
    else create.mutate(body, handlers)
  }

  const live = useWatch({ control })
  const debouncedName = useDebouncedValue((live.name ?? '').trim(), 400)
  const codeIsManual = (live.code ?? '').trim() !== ''
  // Create mode only: ask the server for the code it would generate, until the user types their own.
  const suggestion = useProjectCodeSuggestionQuery(debouncedName, !editing && !codeIsManual)
  const suggestedCode = !editing && !codeIsManual && debouncedName ? suggestion.data?.code : undefined
  const title = editing ? 'Edit project' : 'Create project'
  const sectionCls = 'flex flex-col gap-4'
  const legendCls = 'mb-3 text-xs font-medium text-ink-muted'
  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      noValidate
      aria-label={title}
      className="grid grid-cols-1 items-start gap-6 lg:grid-cols-[minmax(0,1fr)_20rem]"
    >
      <Card padded={false}>
        <div className="flex flex-col gap-6 p-4 sm:p-6">
          <div>
            <h2 className="text-base font-semibold text-ink">{editing ? 'Project details' : 'New project'}</h2>
            <p className="mt-1 text-sm text-ink-muted">
              Fields marked with <span className="text-red-600">*</span> are required.
            </p>
          </div>
          <fieldset className={sectionCls}>
            <legend className={legendCls}>Basics</legend>
            <div className="grid gap-4 sm:grid-cols-2">
              <FormField label="Name" required autoComplete="off" error={errors.name?.message} {...register('name')} />
              <FormField
                label="Code"
                required={editing}
                autoComplete="off"
                placeholder={editing ? undefined : (suggestedCode ?? 'Auto-generated')}
                hint={editing ? 'A short unique identifier, e.g. APOLLO-01.' : 'Auto-generated from the name — you can edit it.'}
                error={errors.code?.message}
                {...register('code')}
              />
              <SelectField label="Status" error={errors.status?.message} {...register('status')}>
                {PROJECT_STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {PROJECT_STATUS_LABELS[s]}
                  </option>
                ))}
              </SelectField>
            </div>
          </fieldset>
          <fieldset className={`${sectionCls} border-t border-line pt-6`}>
            <legend className={legendCls}>Schedule</legend>
            <div className="grid gap-4 sm:grid-cols-2">
              <FormField label="Start date" type="date" required error={errors.start_date?.message} {...register('start_date')} />
              <FormField
                label="End date"
                type="date"
                hint="Leave empty for an ongoing project."
                error={errors.end_date?.message}
                {...register('end_date')}
              />
            </div>
          </fieldset>
          <fieldset className={`${sectionCls} border-t border-line pt-6`}>
            <legend className={legendCls}>Client and description</legend>
            <FormField label="Client name" autoComplete="off" error={errors.client_name?.message} {...register('client_name')} />
            <div className="flex flex-col gap-1">
              <label htmlFor="description" className="text-xs font-medium text-slate-900">
                Description
              </label>
              <textarea
                id="description"
                rows={4}
                className="w-full rounded-sm border border-slate-300 bg-white px-3 py-2 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:text-sm"
                {...register('description')}
              />
            </div>
          </fieldset>
        </div>
      </Card>
      <aside aria-label="Summary" className="flex flex-col gap-4 rounded-sm border border-line bg-surface p-4 lg:sticky lg:top-20">
        <h2 className="text-base font-semibold text-ink">Summary</h2>
        <div className="flex flex-col gap-3">
          <ProjectStatusBadge status={live.status} />
          <dl className="flex flex-col gap-2 text-sm">
            <div>
              <dt className="text-xs font-medium text-ink-muted">Code</dt>
              <dd className="break-words text-ink">{live.code?.trim() || suggestedCode || '-'}</dd>
            </div>
            <div>
              <dt className="text-xs font-medium text-ink-muted">Name</dt>
              <dd className="break-words text-ink">{live.name?.trim() || '-'}</dd>
            </div>
            <div>
              <dt className="text-xs font-medium text-ink-muted">Dates</dt>
              <dd className="text-ink">
                {live.start_date || '-'} to {live.end_date || 'ongoing'}
              </dd>
            </div>
          </dl>
        </div>
        <ul className="list-disc space-y-1 border-t border-line pt-4 pl-4 text-xs text-ink-muted">
          <li>The project code must be unique within your organization.</li>
          <li>The end date is optional; leave it empty for an ongoing project.</li>
          <li>Archived projects cannot receive new allocations.</li>
        </ul>
        <div className="flex flex-col-reverse gap-2 border-t border-line pt-4">
          <Link to={backTo} className={`${secondaryButtonCls} inline-flex items-center justify-center`}>
            Cancel
          </Link>
          <SubmitButton loading={pending}>{editing ? 'Save changes' : 'Create'}</SubmitButton>
        </div>
      </aside>
    </form>
  )
}

export default function ProjectFormPage() {
  const { id } = useParams()
  const { data, isPending, isError, error } = useProjectQuery(id)
  const editing = Boolean(id)
  const backTo = '/projects'
  return (
    <PageContainer>
      <PageHeader
        title={editing ? 'Edit project' : 'Create project'}
        description={editing ? 'Update project details, schedule and status.' : 'Add a project you can allocate employees to.'}
        actions={
          <Link to={backTo} className={`${secondaryButtonCls} inline-flex items-center`}>
            Back to projects
          </Link>
        }
      />
      {editing && isPending && <p role="status" className="text-sm text-slate-600">Loading project…</p>}
      {editing && isError && <InlineAlert tone="error">{getErrorMessage(error, 'Could not load project.')}</InlineAlert>}
      {(!editing || data) && (
        <div className="w-full max-w-5xl">
          <ProjectForm key={data?.id ?? 'new'} project={data} />
        </div>
      )}
    </PageContainer>
  )
}
