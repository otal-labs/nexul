import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { DEFAULTS_DESCRIPTION, PairingDefaultsForm } from "@/components/settings/PairingDefaultsForm";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchPairingDefaults, useListComputers } from "@/hooks/PairingHooks";

export const PairingDefaultsSection = () => {
  const { data: computers, isPending: computersPending, error: computersError } = useListComputers();
  const { data: defaults, isPending: defaultsPending, error: defaultsError } = useFetchPairingDefaults();
  const isPending = computersPending || defaultsPending;
  const error = computersError || defaultsError;

  return (
    <>
      {isPending && (
        <SettingsCard id="pairing-defaults" title="Defaults" description={DEFAULTS_DESCRIPTION}>
          <LoadingDisplay />
        </SettingsCard>
      )}
      {error && (
        <SettingsCard id="pairing-defaults" title="Defaults" description={DEFAULTS_DESCRIPTION}>
          <ErrorDisplay error={error} />
        </SettingsCard>
      )}
      {computers && defaults && <PairingDefaultsForm defaults={defaults} computers={computers} />}
    </>
  );
};
