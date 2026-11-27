import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PaneHeader } from "@/components/computers/PaneHeader";
import { HarnessReadinessLine } from "@/components/settings/HarnessReadinessLine";
import { PairingDefaultsForm } from "@/components/settings/PairingDefaultsForm";
import { useFetchPairingDefaults } from "@/hooks/PairingHooks";

// What @Agent and every project's runs use until a project changes it, and where a chat outside a project starts now.
export const DefaultsRunPane = () => {
  const { data: defaults, error, isPending } = useFetchPairingDefaults();

  return (
    <div className="space-y-6">
      <PaneHeader title="Defaults" meta={<HarnessReadinessLine />} />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {defaults && (
        <div className="@container max-w-2xl">
          <PairingDefaultsForm defaults={defaults} />
        </div>
      )}
    </div>
  );
};
