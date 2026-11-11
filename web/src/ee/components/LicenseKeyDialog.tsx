import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { toast } from "sonner";

import { errorMessage } from "@/api/client";
import { ActionButton } from "@/components/action/ActionButton";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { useSetLicense } from "@/ee/hooks/LicenseHooks";
import { LicenseKeyFormSchema, type LicenseKeyFormData } from "@/ee/models/License";

interface LicenseKeyDialogProps {
  replacing: boolean;
}

// The key is checked on this instance only: the server refuses one for another address, a bad signature or a passed expiry.
export const LicenseKeyDialog = ({ replacing }: LicenseKeyDialogProps) => {
  const [open, setOpen] = useState(false);
  const setLicense = useSetLicense();
  const form = useForm<LicenseKeyFormData>({ defaultValues: { key: "" }, resolver: zodResolver(LicenseKeyFormSchema) });

  const close = (next: boolean) => {
    setOpen(next);
    if (!next) form.reset();
  };

  const onSave = async ({ key }: LicenseKeyFormData) => {
    try {
      await setLicense.mutateAsync(key);
      toast.success("License key saved");
      close(false);
    } catch (err) {
      form.setError("key", { message: errorMessage(err) });
    }
  };

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogTrigger asChild>
        <ActionButton
          permission="instance:write"
          verb="change the license key"
          size="sm"
          variant={replacing ? "outline" : "default"}
          className="ml-auto"
        >
          {replacing ? "Replace key" : "Add license key"}
        </ActionButton>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{replacing ? "Replace the license key" : "Add a license key"}</DialogTitle>
          <DialogDescription>Paste the whole key you were sent. It starts with nexul-license-1.</DialogDescription>
        </DialogHeader>
        <form onSubmit={form.handleSubmit(onSave)} className="space-y-4">
          <Controller
            control={form.control}
            name="key"
            render={({ field, fieldState }) => (
              <div className="space-y-1.5">
                <Textarea
                  {...field}
                  aria-label="License key"
                  aria-invalid={!!fieldState.error}
                  placeholder="nexul-license-1."
                  autoComplete="off"
                  spellCheck={false}
                  data-1p-ignore="true"
                  data-lpignore="true"
                  className="h-32 font-mono text-xs break-all"
                />
                {fieldState.error && <p className="text-sm text-destructive">{fieldState.error.message}</p>}
              </div>
            )}
          />
          <DialogFooter>
            <Button type="submit" loading={form.formState.isSubmitting}>
              Save key
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
