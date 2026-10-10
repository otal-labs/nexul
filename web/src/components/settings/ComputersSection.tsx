import { PlusIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { AddComputerDialog } from "@/components/pairing/AddComputerDialog";
import { ComputerItem } from "@/components/settings/ComputerItem";
import { HarnessReadinessLine } from "@/components/settings/HarnessReadinessLine";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchPresence, useListComputers } from "@/hooks/PairingHooks";

interface ComputersSectionProps {
  // The owner wizard frames the title itself and is where the first computer is added, so it drops the card and the readiness line.
  bare?: boolean;
}

export const ComputersSection = ({ bare = false }: ComputersSectionProps = {}) => {
  const { data: computers, isPending, error } = useListComputers();
  const presence = useFetchPresence();

  const addButton = (
    <AddComputerDialog
      trigger={
        <Button type="button" size={bare ? "default" : "sm"}>
          <PlusIcon className="size-4" aria-hidden />
          Add a computer
        </Button>
      }
    />
  );
  const list = (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {computers && computers.length === 0 && <EmptyRow flush>No computers yet</EmptyRow>}
      {computers && computers.length > 0 && (
        <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
          {computers.map((computer) => (
            <ComputerItem key={computer.id} computer={computer} presence={presence.data?.[computer.id]} />
          ))}
        </EnterList>
      )}
    </>
  );

  return (
    <>
      {bare && (
        <div className="space-y-4">
          <div className="flex flex-wrap gap-2">{addButton}</div>
          {list}
        </div>
      )}
      {!bare && (
        <SettingsCard
          id="pairing-computers"
          title="Your computers"
          description="The computers @Agent works through, each running T3 Code and the Nexul app. Nexul keeps them paired."
          footer={addButton}
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
