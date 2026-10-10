import { createContext, useContext } from "react";
import type { FieldValues, UseFormReturn } from "react-hook-form";
import { useFormContext } from "react-hook-form";

// A sub-form shown inside the dialog (an auto play opened from a play) takes over the footer while it is open.
export interface FooterIntercept {
  submit: () => Promise<void>;
  cancel: () => void;
}

export interface FormDialogContextValue<TFormValues extends FieldValues = FieldValues> {
  setLoading: (loading: boolean) => void;
  onSubmit: (handler: (data: TFormValues) => Promise<TFormValues>) => void;
  // "Create more": a form flags via setStayOpen that submit should keep the dialog open, then cleans up itself.
  setStayOpen: (stayOpen: boolean) => void;
  onAfterSubmit: (handler: () => void) => void;
  // Same submit path as the OK button, so a form can wire its own shortcut (e.g. Cmd/Ctrl+Enter).
  submit: () => void;
  // While set, OK, Cancel, Esc and the close button act on the sub-form instead of the dialog's own form; null hands them back.
  intercept: (handlers: FooterIntercept | null) => void;
}

// Context can't be parameterized, so the value is type-erased; useFormDialogContext<T>() restores it.
export const FormDialogContext = createContext<FormDialogContextValue<FieldValues> | null>(null);

export const useFormDialogContext = <TFormValues extends FieldValues>(): UseFormReturn<TFormValues> &
  FormDialogContextValue<TFormValues> => {
  const dialogContext = useContext(FormDialogContext);
  if (dialogContext === null) {
    throw new Error("useFormDialogContext must be used within a FormDialog");
  }
  const rhfMethods = useFormContext<TFormValues>();
  return { ...rhfMethods, ...(dialogContext as FormDialogContextValue<TFormValues>) };
};
