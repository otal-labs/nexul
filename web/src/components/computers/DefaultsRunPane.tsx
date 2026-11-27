import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PaneHeader } from "@/components/computers/PaneHeader";
import { HarnessReadinessLine } from "@/components/settings/HarnessReadinessLine";
import { PairingDefaultsForm } from "@/components/settings/PairingDefaultsForm";
import { useFetchPairingDefaults } from "@/hooks/PairingHooks";
import { useFlash } from "@/hooks/useFlash";

// What @Agent and every project's runs use until a project changes it, and where a chat outside a project starts now.
export const DefaultsRunPane = () => {
  const { data: defaults, error, isPending } = useFetchPairingDefaults();
  const [saved, flash] = useFlash();

  return (
    <div className="space-y-6">
      <PaneHeader title="Defaults" meta={<HarnessReadinessLine />} note={saved ? "Saved" : "Changes save as you go"} />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {defaults && (
        <div className="@container max-w-2xl">
          <PairingDefaultsForm defaults={defaults} onSaved={flash} />
        </div>
      )}
    </div>
  );
};
