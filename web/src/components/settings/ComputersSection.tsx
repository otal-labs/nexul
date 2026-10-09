import { PlusIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PairComputerDialog } from "@/components/pairing/PairComputerDialog";
import { ComputerRow } from "@/components/settings/ComputerRow";
import { HarnessReadinessLine } from "@/components/settings/HarnessReadinessLine";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchPresence, useListComputers } from "@/hooks/PairingHooks";

interface ComputersSectionProps {
  // The owner wizard frames the title itself and is where pairing happens, so it drops the card and the readiness line.
  bare?: boolean;
}

export const ComputersSection = ({ bare = false }: ComputersSectionProps = {}) => {
  const { data: computers, isPending, error } = useListComputers();
  const presence = useFetchPresence();

  const pairButton = (
    <PairComputerDialog
      trigger={
        <Button type="button" size={bare ? "default" : "sm"}>
          <PlusIcon className="size-4" aria-hidden />
          Pair a computer
        </Button>
      }
    />
  );
  const list = (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {computers && computers.length === 0 && <EmptyRow flush>No computers paired yet</EmptyRow>}
      {computers && computers.length > 0 && (
        <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
          {computers.map((computer) => (
            <ComputerRow key={computer.id} computer={computer} presence={presence.data?.[computer.id]} />
          ))}
        </EnterList>
      )}
    </>
  );

  return (
    <>
      {bare && (
        <div className="space-y-4">
          <div className="flex flex-wrap gap-2">{pairButton}</div>
          {list}
        </div>
      )}
      {!bare && (
        <SettingsCard
          id="pairing-computers"
          title="Paired computers"
          description="Computers running T3 Code that @Agent works through. A pairing lasts 30 days and can't renew itself, so re-pair before it runs out."
          footer={pairButton}
        >
          <div className="space-y-4">
            <HarnessReadinessLine />
            {list}
          </div>
        </SettingsCard>
      )}
    </>
  );
};
