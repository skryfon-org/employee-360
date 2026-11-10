import { useRef } from "react";
import { Controller, useForm, useWatch } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { getErrorMessage, isConflictError } from "@employee360/api-client";
import { FormField, SubmitButton, Switch } from "@employee360/ui";
import { useDialogFocus } from "../../../hooks/useDialogFocus";
import { useToast } from "../../../hooks/useToast";
import {
  useCreateLeaveTypeMutation,
  useUpdateLeaveTypeMutation,
  type LeaveType,
} from "../queries/leaveQueries";
import {
  DESCRIPTION_MAX,
  leaveTypeFormSchema,
  type LeaveTypeFormValues,
} from "../schemas/leaveTypeSchemas";
import { FIELD, LABEL, SECONDARY_BTN } from "./styles";

export function LeaveTypeFormModal({
  leaveType,
  onClose,
}: {
  leaveType?: LeaveType;
  onClose: () => void;
}) {
  const toast = useToast();
  const create = useCreateLeaveTypeMutation();
  const update = useUpdateLeaveTypeMutation();
  const editing = Boolean(leaveType);
  const pending = create.isPending || update.isPending;
  const dialogRef = useRef<HTMLDivElement>(null);
  const cancelRef = useRef<HTMLButtonElement>(null);
  useDialogFocus(dialogRef, cancelRef, onClose, pending);

  const {
    register,
    handleSubmit,
    setError,
    setFocus,
    control,
    formState: { errors },
  } = useForm<LeaveTypeFormValues>({
    resolver: zodResolver(leaveTypeFormSchema),
    defaultValues: {
      name: leaveType?.name ?? "",
      code: leaveType?.code ?? "",
      description: leaveType?.description ?? "",
      is_active: leaveType?.is_active ?? true,
    },
  });
  const descLength = [...(useWatch({ control, name: "description" }) ?? "")]
    .length;

  const onSubmit = (v: LeaveTypeFormValues) => {
    const handlers = {
      onSuccess: () => {
        toast.success(
          editing
            ? `Leave type "${v.name}" updated.`
            : `Leave type "${v.name}" created.`,
        );
        onClose();
      },
      onError: (err: unknown) => {
        if (isConflictError(err)) {
          setError("code", {
            type: "server",
            message: "A leave type with this code already exists.",
          });
          setFocus("code");
          return;
        }
        toast.error(
          getErrorMessage(
            err,
            editing
              ? "Could not update leave type."
              : "Could not create leave type.",
          ),
        );
      },
    };
    if (leaveType) update.mutate({ id: leaveType.id, body: v }, handlers);
    else create.mutate(v, handlers);
  };

  const title = editing ? "Edit leave type" : "Add leave type";
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="leave-type-form-title"
        className="max-h-[calc(100dvh-2rem)] w-full max-w-md overflow-y-auto rounded-sm border border-slate-300 bg-white p-4 sm:p-6"
      >
        <h2
          id="leave-type-form-title"
          className="text-base font-semibold text-slate-900"
        >
          {title}
        </h2>
        <form
          onSubmit={handleSubmit(onSubmit)}
          noValidate
          aria-label={title}
          className="mt-4 flex flex-col gap-4"
        >
          <FormField
            label="Name"
            autoComplete="off"
            error={errors.name?.message}
            {...register("name")}
          />
          <FormField
            label="Code"
            autoComplete="off"
            error={errors.code?.message}
            {...register("code")}
          />
          <div className="flex flex-col gap-1">
            <label htmlFor="leave-type-description" className={LABEL}>
              Description (optional)
            </label>
            <textarea
              id="leave-type-description"
              rows={3}
              aria-invalid={errors.description ? true : undefined}
              className={`${FIELD} h-auto py-2 md:h-auto`}
              {...register("description")}
            />
            {errors.description ? (
              <p role="alert" className="text-xs text-red-700">
                {errors.description.message}
              </p>
            ) : (
              <p className="text-xs text-slate-600">
                {descLength}/{DESCRIPTION_MAX} characters.
              </p>
            )}
          </div>
          <Controller
            control={control}
            name="is_active"
            render={({ field }) => (
              <Switch
                id="leave-type-is-active"
                ref={field.ref}
                name={field.name}
                checked={field.value}
                onChange={field.onChange}
                onBlur={field.onBlur}
                label="Active"
                description="Inactive types cannot be chosen for new requests; existing requests are kept."
              />
            )}
          />
          <div className="mt-2 flex justify-end gap-2">
            <button
              ref={cancelRef}
              type="button"
              onClick={onClose}
              disabled={pending}
              className={SECONDARY_BTN}
            >
              Cancel
            </button>
            <SubmitButton loading={pending}>
              {editing ? "Save changes" : "Add leave type"}
            </SubmitButton>
          </div>
        </form>
      </div>
    </div>
  );
}
