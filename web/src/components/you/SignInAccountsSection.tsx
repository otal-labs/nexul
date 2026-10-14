import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { IdentityRow } from "@/components/you/IdentityRow";
import { useBootstrapStatus, useFetchIdentities } from "@/hooks/AuthHooks";
import { useLinkResultToast } from "@/hooks/useLinkResultToast";
import type { BootstrapStatus, Provider } from "@/models/User";

// Only providers the instance has turned on are offered; GitHub is on once bootstrap stored its App.
const enabledProviders = (status: BootstrapStatus): Provider[] => [
  ...(status.configured ? (["github"] as const) : []),
  ...(status.google_configured ? (["google"] as const) : []),
  ...(status.discord_configured ? (["discord"] as const) : []),
];

export const SignInAccountsSection = () => {
  const identities = useFetchIdentities();
  const status = useBootstrapStatus();
  useLinkResultToast();

  const isPending = identities.isPending || status.isPending;
  const error = identities.error ?? status.error;

  return (
    <SettingsCard
      id="sign-in-accounts"
      title="Sign-in accounts"
      description="Each linked account signs you in to this profile."
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
