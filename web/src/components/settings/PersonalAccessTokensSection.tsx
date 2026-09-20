import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { Check, Copy } from "lucide-react";
import { useForm } from "react-hook-form";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { PATRow } from "@/components/settings/PATRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useListPATs, useMintPAT } from "@/hooks/AuthHooks";
import { PATNameFormSchema, type PATNameFormData } from "@/models/User";

export const PersonalAccessTokensSection = () => {
  const { data, isPending, error } = useListPATs();
  const mint = useMintPAT();
  const [newToken, setNewToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const form = useForm<PATNameFormData>({
    defaultValues: { name: "" },
    resolver: zodResolver(PATNameFormSchema),
  });

  const onSubmit = async (data: PATNameFormData) => {
    try {
      const res = await mint.mutateAsync(data.name);
      form.reset();
      setNewToken(res.token);
      setCopied(false);
    } catch {
      // Error is surfaced by the hook's toast; the form stays open to retry.
    }
  };

  const copyToken = async () => {
    if (!newToken) return;
    try {
      await navigator.clipboard.writeText(newToken);
      setCopied(true);
    } catch {
      // Clipboard may be unavailable; the box stays visible for manual copy.
    }
  };

  return (
    <SettingsCard
      id="personal-tokens"
      title="Personal access tokens"
      description="Long-lived credentials agents and personal integrations use to talk to this instance. They
        act as you with your full permissions. The raw token is shown once when created — it can
        never be listed again. Revoke one to cut its access immediately."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && (
        <div className="space-y-6">
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-wrap items-end gap-2">
            <FormInput
              control={form.control}
              name="name"
              id="token-name"
              label="Token name"
              placeholder="e.g. ci agent"
              className="w-full sm:w-80"
            />
            <Button type="submit" disabled={mint.isPending}>
              {mint.isPending ? "Creating…" : "Create token"}
            </Button>
          </form>

          {newToken && (
            <div className="animate-in fade-in-0 slide-in-from-top-1 space-y-2 rounded-md bg-muted p-4 duration-200 ease-out">
              <p className="text-sm font-medium">Copy this token now — it won't be shown again.</p>
              <p className="break-all rounded bg-card p-2 font-mono text-xs">{newToken}</p>
              <Button type="button" variant="outline" size="sm" onClick={copyToken}>
                {copied && <Check className="size-4" />}
                {!copied && <Copy className="size-4" />}
                {copied ? "Copied" : "Copy"}
              </Button>
            </div>
          )}

          {data.tokens.length === 0 && (
            <NoDataDisplay message="No personal access tokens yet" />
          )}
          {data.tokens.length > 0 && (
            <ul className="divide-y divide-border overflow-hidden rounded-md border">
              {data.tokens.map((token) => (
                <PATRow key={token.id} token={token} />
              ))}
            </ul>
          )}
        </div>
      )}
    </SettingsCard>
  );
};
