import { SlidersHorizontalIcon } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { PairingDefaultsForm } from "@/components/settings/PairingDefaultsForm";
import { RunSettingsRow } from "@/components/settings/RunSettingsRow";
import { useFetchPairingDefaults } from "@/hooks/PairingHooks";
import { useDefaultsSummary } from "@/hooks/PairingProjectHooks";

interface PairingDefaultsRowProps {
  open: boolean;
  onToggle: () => void;
}

// The list's first row: what every project's runs use until the project changes it.
export const PairingDefaultsRow = ({ open, onToggle }: PairingDefaultsRowProps) => {
  const { data: defaults, error } = useFetchPairingDefaults();
  const summary = useDefaultsSummary(defaults);

  return (
    <RunSettingsRow
      mark={
        <span aria-hidden className="flex size-7 shrink-0 items-center justify-center rounded-md bg-muted/60 text-muted-foreground">
          <SlidersHorizontalIcon className="size-3.5" />
        </span>
      }
      title="Defaults"
      summary={error ? "Couldn't load" : summary}
      open={open}
      onToggle={onToggle}
    >
      {error && <ErrorDisplay error={error} />}
      {defaults && <PairingDefaultsForm defaults={defaults} />}
    </RunSettingsRow>
  );
};
