import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { CopyButton } from "@/components/settings/CopyButton";
import { PATRow } from "@/components/settings/PATRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useListPATs, useMintPAT } from "@/hooks/AuthHooks";
import { PATNameFormSchema, type PATNameFormData } from "@/models/User";

export const PersonalAccessTokensSection = () => {
  const { data, isPending, error } = useListPATs();
  const mint = useMintPAT();
  const [newToken, setNewToken] = useState<string | null>(null);

  const form = useForm<PATNameFormData>({
    defaultValues: { name: "" },
    resolver: zodResolver(PATNameFormSchema),
  });

  const onSubmit = async (data: PATNameFormData) => {
    try {
      const res = await mint.mutateAsync(data.name);
      form.reset();
      setNewToken(res.token);
    } catch {
      // Error is surfaced by the hook's toast; the form stays open to retry.
    }
  };

  return (
    <SettingsCard
      id="personal-tokens"
      title="Personal access tokens"
      description="For scripts and agents that call this instance as you, with all your permissions. Revoking one cuts it off at once."
      footer={
        <form onSubmit={form.handleSubmit(onSubmit)} className="flex w-full flex-wrap items-start gap-2">
          <div className="min-w-0 flex-1 basis-56">
            <FormInput control={form.control} name="name" id="token-name" label="Token name" hideLabel placeholder="Token name, such as CI agent" />
          </div>
          <Button type="submit" variant="outline" loading={mint.isPending}>
            Create token
          </Button>
        </form>
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && (
        <div className="space-y-4">
          {newToken && (
            <div className="animate-in fade-in-0 slide-in-from-top-1 space-y-2 rounded-md border border-border bg-surface-2 p-3 duration-200 ease-out">
              <p className="text-sm font-medium">Copy this token now. It won't be shown again.</p>
              <div className="flex items-center gap-2 rounded-md bg-card px-2 py-1.5 ring-1 ring-border">
                <code className="min-w-0 flex-1 font-mono text-xs break-all">{newToken}</code>
                <CopyButton value={newToken} label="Copy token" iconOnly variant="ghost" />
              </div>
            </div>
          )}
          {data.tokens.length === 0 && <EmptyRow flush>No personal access tokens yet. Name one below to create it.</EmptyRow>}
          {data.tokens.length > 0 && (
            <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
              {data.tokens.map((token) => (
                <PATRow key={token.id} token={token} />
              ))}
            </EnterList>
          )}
        </div>
      )}
    </SettingsCard>
  );
};
