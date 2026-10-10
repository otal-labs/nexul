import { zodResolver } from "@hookform/resolvers/zod";
import type { ComponentType, ReactNode } from "react";
import { createElement, useRef, useState } from "react";
import type { ConfirmDialogProps as ReactConfirmDialogProps } from "react-confirm";
import type { FieldValues, UseFormProps } from "react-hook-form";
import { FormProvider, useForm } from "react-hook-form";
import type { ZodType } from "zod";

import { errorMessage } from "@/api/client";
import { FormDialogContext, type FooterIntercept } from "@/components/dialogs/FormDialogContext";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export interface FormDialogResult<TFormValues extends FieldValues = FieldValues> {
  success: boolean;
  data: TFormValues | null;
}

export interface FormDialogProps<TFormValues extends FieldValues = FieldValues> {
  title: string;
  description?: string;
  // Replaces the title text; `title` still renders as a visually-hidden DialogTitle for accessibility.
  header?: ReactNode;
  // Rendered left of Cancel/OK in DialogFooter (e.g. a "Create more" checkbox); the buttons stay right-aligned.
  footerStart?: ReactNode;
  form: ReactNode | ComponentType;
  schema: ZodType<TFormValues, TFormValues>;
  okLabel?: string;
  cancelLabel?: string;
  formOptions?: Omit<UseFormProps<TFormValues>, "resolver">;
  // Scopes a token override to one caller's DialogContent without touching every other FormDialog.
  dialogClassName?: string;
}

export const FormDialog = ({
  show,
  proceed,
  title,
  description,
  header,
  footerStart,
  form,
  schema,
  okLabel = "Save",
  cancelLabel = "Cancel",
  formOptions,
  dialogClassName,
}: ReactConfirmDialogProps<FormDialogProps, FormDialogResult>) => {
  const [isLoading, setIsLoading] = useState(false);
  const submitHandlerRef = useRef<((data: FieldValues) => Promise<FieldValues>) | null>(null);
  // "Create more": a form can flag via context to keep the dialog open and clean itself up via onAfterSubmit.
  const stayOpenRef = useRef(false);
  const afterSubmitHandlerRef = useRef<(() => void) | null>(null);
  const interceptRef = useRef<FooterIntercept | null>(null);
  const [intercepting, setIntercepting] = useState(false);

  const methods = useForm<FieldValues>({
    ...formOptions,
    resolver: zodResolver(schema),
  });

  const handleOkClick = async (data: FieldValues): Promise<void> => {
    if (submitHandlerRef.current === null) {
      proceed({ success: true, data });
      return;
    }
    methods.clearErrors("root.serverError");
    try {
      const result = await submitHandlerRef.current(data);
      if (stayOpenRef.current) {
        afterSubmitHandlerRef.current?.();
        return;
      }
      proceed({ success: true, data: result });
    } catch (error) {
      if (Object.keys(methods.formState.errors).length === 0) {
        methods.setError("root.serverError", {
          type: "server",
          // errorMessage reads the backend envelope; Error.message alone is a useless status-code string.
          message: errorMessage(error),
        });
      }
    }
  };

  const handleAction = async (): Promise<void> => {
    const intercept = interceptRef.current;
    if (intercept) {
      setIntercepting(true);
      await intercept.submit().finally(() => setIntercepting(false));
      return;
    }
    await methods.handleSubmit(handleOkClick)();
  };

  const handleCancel = (): void => {
    if (interceptRef.current) {
      interceptRef.current.cancel();
      return;
    }
    proceed({ success: false, data: null });
  };

  const contextValue = {
    onSubmit: (handler: (data: FieldValues) => Promise<FieldValues>) => {
      submitHandlerRef.current = handler;
    },
    setLoading: setIsLoading,
    setStayOpen: (stayOpen: boolean) => {
      stayOpenRef.current = stayOpen;
    },
    onAfterSubmit: (handler: () => void) => {
      afterSubmitHandlerRef.current = handler;
    },
    submit: () => {
      void handleAction();
    },
    intercept: (handlers: FooterIntercept | null) => {
      interceptRef.current = handlers;
    },
  };

  const busy = isLoading || intercepting || methods.formState.isSubmitting;

  return (
    <Dialog
      open={show}
      onOpenChange={(open) => {
        if (!open) handleCancel();
      }}
    >
      <DialogContent
        className={cn(dialogClassName)}
        // A stray click beside the dialog must not throw typed input away; Cancel and Esc close it on purpose.
        onInteractOutside={(e) => e.preventDefault()}
      >
        <FormProvider {...methods}>
          <FormDialogContext.Provider value={contextValue}>
            <DialogHeader>
              {header ? (
                <>
                  <DialogTitle className="sr-only">{title}</DialogTitle>
                  {header}
                </>
              ) : (
                <DialogTitle>{title}</DialogTitle>
              )}
              {description !== undefined && <DialogDescription>{description}</DialogDescription>}
            </DialogHeader>
            <DialogBody className="flex flex-col gap-4">
              {typeof form === "function" ? createElement(form) : form}
              {methods.formState.errors.root?.serverError && (
                <p role="alert" className="text-sm text-destructive">
                  {methods.formState.errors.root.serverError.message}
                </p>
              )}
            </DialogBody>
            <DialogFooter className={cn(footerStart && "sm:justify-between")}>
              {footerStart}
              <div className="flex flex-col-reverse gap-2 sm:flex-row">
                <Button variant="outline" onClick={handleCancel}>
                  {cancelLabel}
                </Button>
                <Button onClick={handleAction} loading={busy}>
                  {okLabel}
                </Button>
              </div>
            </DialogFooter>
          </FormDialogContext.Provider>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
};
