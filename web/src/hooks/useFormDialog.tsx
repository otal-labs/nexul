import { ContextAwareConfirmation, confirmable } from "react-confirm";
import type { FieldValues } from "react-hook-form";

import {
  FormDialog,
  type FormDialogProps,
  type FormDialogResult,
} from "@/components/dialogs/FormDialog";

const formConfirm = ContextAwareConfirmation.createConfirmation<FormDialogProps, FormDialogResult>(
  confirmable<FormDialogProps, FormDialogResult>(FormDialog),
  // 0 = unmount as soon as it settles; the default delay leaves a stale entry a re-mount would resurrect.
  0,
);

export const useFormDialog = () => ({
  // The shell's types are fixed at createConfirmation time; per-call TFormValues is asserted for ergonomics.
  open: <TFormValues extends FieldValues>(
    props: FormDialogProps<TFormValues>,
  ): Promise<FormDialogResult<TFormValues>> =>
    formConfirm(props as FormDialogProps<FieldValues>) as Promise<
      FormDialogResult<TFormValues>
    >,
});
