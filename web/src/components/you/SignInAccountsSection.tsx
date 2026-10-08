import { useEffect, useRef } from "react";
import { useSearchParams } from "react-router";
import { toast } from "sonner";

import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { IdentityRow } from "@/components/you/IdentityRow";
import { useBootstrapStatus, useFetchIdentities } from "@/hooks/AuthHooks";
import { providerLabel, type BootstrapStatus, type Provider } from "@/models/User";

// Only providers the instance has turned on are offered; GitHub is on once bootstrap stored its App.
const enabledProviders = (status: BootstrapStatus): Provider[] => [
  ...(status.configured ? (["github"] as const) : []),
  ...(status.google_configured ? (["google"] as const) : []),
  ...(status.discord_configured ? (["discord"] as const) : []),
];

const isProvider = (value: string | null): value is Provider => value !== null && value in providerLabel;

export const SignInAccountsSection = () => {
  const identities = useFetchIdentities();
  const status = useBootstrapStatus();
  const [searchParams, setSearchParams] = useSearchParams();
  const toasted = useRef(false);

  // Toasts the link callback once, then strips its params so a refresh doesn't re-fire it.
  useEffect(() => {
    if (toasted.current) return;
    const provider = searchParams.get("provider");
    const error = searchParams.get("error");
    if (!isProvider(provider) || (searchParams.get("linked") !== "1" && !error)) return;
    toasted.current = true;
    if (error) toast.error(`${providerLabel[provider]}: ${error}`);
    if (!error) toast.success(`${providerLabel[provider]} linked`);
    const next = new URLSearchParams(searchParams);
    next.delete("provider");
    next.delete("linked");
    next.delete("error");
    setSearchParams(next, { replace: true });
  }, [searchParams, setSearchParams]);

  const isPending = identities.isPending || status.isPending;
  const error = identities.error ?? status.error;

  return (
    <SettingsCard
      id="sign-in-accounts"
      title="Sign-in accounts"
      description="Any linked account signs you in to this same profile."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {identities.data && status.data && (
        <EnterList className="divide-y divide-border overflow-hidden rounded-md border border-border">
          {enabledProviders(status.data).map((provider) => (
            <IdentityRow
              key={provider}
              provider={provider}
              identity={identities.data.find((identity) => identity.provider === provider)}
              onlyOne={identities.data.length === 1}
            />
          ))}
        </EnterList>
      )}
    </SettingsCard>
  );
};
