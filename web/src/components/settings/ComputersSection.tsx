import { PairComputerDialog } from "@/components/pairing/PairComputerDialog";
import { ComputerRow } from "@/components/settings/ComputerRow";
import { HarnessReadinessLine } from "@/components/settings/HarnessReadinessLine";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { useFetchPresence, useListComputers } from "@/hooks/PairingHooks";

export const ComputersSection = () => {
  const { data: computers, isPending, error } = useListComputers();
  const presence = useFetchPresence();

  return (
    <SettingsCard
      id="pairing-computers"
      title="Paired computers"
      description="Every computer running T3 Code that @Agent can act through on your behalf. Pairing trades a
        one-time `t3 pair` token for a 30-day bearer session — there's no auto-refresh upstream, so
        re-pair before it expires or the computer starts acting exactly like unpaired."
    >
      <div className="space-y-4">
        <HarnessReadinessLine />
        <div className="flex flex-wrap gap-2">
          <PairComputerDialog />
        </div>
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} />}
        {computers && computers.length === 0 && <NoDataDisplay message="No computers paired yet" />}
        {computers && computers.length > 0 && (
          <ul className="divide-y divide-border overflow-hidden rounded-md border">
            {computers.map((computer) => (
              <ComputerRow key={computer.id} computer={computer} presence={presence.data?.[computer.id]} />
            ))}
          </ul>
        )}
      </div>
    </SettingsCard>
  );
};
