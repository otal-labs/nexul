import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PairingDefaultsForm } from "@/components/settings/PairingDefaultsForm";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchPairingDefaults, useListComputers } from "@/hooks/PairingHooks";

export const PairingDefaultsSection = () => {
  const { data: computers, isPending: computersPending, error: computersError } = useListComputers();
  const { data: defaults, isPending: defaultsPending, error: defaultsError } = useFetchPairingDefaults();
  const isPending = computersPending || defaultsPending;
  const error = computersError || defaultsError;

  return (
    <SettingsCard
      id="pairing-defaults"
      title="Defaults"
      description="Used by @Agent in channels and DMs outside a project, and in every project you haven't linked
        on the Projects tab."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {computers && defaults && <PairingDefaultsForm defaults={defaults} computers={computers} />}
    </SettingsCard>
  );
};
