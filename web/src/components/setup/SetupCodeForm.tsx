import { zodResolver } from "@hookform/resolvers/zod";
import type { AxiosError } from "axios";
import { useEffect } from "react";
import { useForm } from "react-hook-form";

import { errorMessage, type ApiErrorBody } from "@/api/client";
import { FormInput } from "@/components/FormInput";
import { CommandRow } from "@/components/pairing/CommandRow";
import { Button } from "@/components/ui/button";
import { useUnlockSetup } from "@/hooks/SetupHooks";
import { SetupCodeFormSchema, codeFromFragment, type SetupCodeFormData } from "@/models/Setup";

const unlockError = (error: unknown): string => {
  const response = (error as AxiosError<ApiErrorBody>).response;
  if (response?.status === 429) return "Too many wrong codes from this address. Wait a few minutes, then try again.";
  if (response?.status === 409) return "Setup is already done on this instance. Sign in instead.";
  if (response?.data?.code === "invalid_code") return "That code is wrong or has expired. Check it and try again.";
  return errorMessage(error);
};

// The handoff link and the installer can carry the code in #code=, which stays out of server logs.
export const SetupCodeForm = () => {
  const unlock = useUnlockSetup();
  const form = useForm<SetupCodeFormData>({
    defaultValues: { code: codeFromFragment(window.location.hash) },
    resolver: zodResolver(SetupCodeFormSchema),
  });

  // One-shot: drop the code from the address bar once the form has read it, so it isn't bookmarked or shared.
  useEffect(() => {
    if (!window.location.hash) return;
    window.history.replaceState(null, "", `${window.location.pathname}${window.location.search}`);
  }, []);

  const onSubmit = async ({ code }: SetupCodeFormData) => {
    try {
      await unlock.mutateAsync(code);
    } catch (err) {
      form.setError("code", { message: unlockError(err) });
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
      <FormInput
        control={form.control}
        name="code"
        label="Setup code"
        placeholder="nxs_…"
        autoComplete="off"
        spellCheck={false}
        className="font-mono"
      />
      <div className="space-y-2">
        <p className="text-[13px] text-muted-foreground">The installer printed it. Lost it? Run this on the server:</p>
        <CommandRow lines={["sudo nexul status"]} ariaLabel="The command to run on the server" copyLabel="command" quiet />
      </div>
      <Button type="submit" size="lg" className="w-full" loading={form.formState.isSubmitting}>
        Continue
      </Button>
    </form>
  );
};
