import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { useActionState } from "@/hooks/useActionState";
import { useFlash } from "@/hooks/useFlash";
import { useUpdateBranding } from "@/ee/hooks/BrandingHooks";
import { BrandingFormSchema, type Branding, type BrandingFormData } from "@/ee/models/Branding";

// The name, the accent and the CSS save together from the card's footer; a logo and a theme save the moment they are chosen.
export const useBrandingForm = (branding: Branding) => {
  const update = useUpdateBranding();
  const [saved, flash] = useFlash();
  const writable = useActionState({ permission: "instance:write" }).state === "live";
  const disabled = !writable || !branding.unlocked;
  const form = useForm<BrandingFormData>({
    disabled,
    // values, not defaultValues: a change an agent saves while the card is open reaches the fields, so a later save keeps it.
    values: { product_name: branding.product_name, accent: branding.accent, css: branding.css },
    resetOptions: { keepDirtyValues: true },
    resolver: zodResolver(BrandingFormSchema),
  });

  const onSubmit = async (data: BrandingFormData) => {
    try {
      await update.mutateAsync(data);
      form.reset(data);
      flash();
    } catch {
      // The hook's toast says why; the form stays open to retry.
    }
  };

  return { form, disabled, saved, onSubmit: form.handleSubmit(onSubmit) };
};
