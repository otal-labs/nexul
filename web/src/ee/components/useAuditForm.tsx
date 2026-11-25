import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { useActionState } from "@/hooks/useActionState";
import { useFlash } from "@/hooks/useFlash";
import { useSaveAuditSettings } from "@/ee/hooks/AuditLogHooks";
import { AuditFormSchema, auditFormValues, auditPatchOf, type AuditFormData, type AuditSettings } from "@/ee/models/AuditLog";

// The days, the destination, its secret and the switch save together from the card's footer.
export const useAuditForm = (settings: AuditSettings) => {
  const save = useSaveAuditSettings();
  const [saved, flash] = useFlash();
  const writable = useActionState({ permission: "instance:write" }).state === "live";
  const disabled = !writable || !settings.unlocked;
  const form = useForm<AuditFormData>({
    disabled,
    // values, not defaultValues: a change an agent saves while the card is open reaches the fields.
    values: auditFormValues(settings),
    resetOptions: { keepDirtyValues: true },
    resolver: zodResolver(AuditFormSchema),
  });

  const onSubmit = async (data: AuditFormData) => {
    try {
      await save.mutateAsync(auditPatchOf(data));
      form.reset({ ...data, stream_secret: "" });
      flash();
    } catch {
      // The hook's toast says why; the form stays as typed to retry.
    }
  };

  return { form, disabled, saved, madeSecret: save.data?.new_secret, onSubmit: form.handleSubmit(onSubmit) };
};
