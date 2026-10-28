import { PlusIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { AddComputerDialog } from "@/components/pairing/AddComputerDialog";
import { ComputerOrderList } from "@/components/settings/ComputerOrderList";
import { OpenComputerCodes } from "@/components/settings/OpenComputerCodes";
import { HarnessReadinessLine } from "@/components/settings/HarnessReadinessLine";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SharedComputersSection } from "@/components/settings/SharedComputersSection";
import { useListComputers } from "@/hooks/PairingHooks";

export const ComputersSection = () => {
  const { data: computers, isPending, error } = useListComputers();

  return (
    <SettingsCard
      id="pairing-computers"
      title="Run order"
      description="Runs go to the first ready computer. Drag to reorder."
      footer={
        <AddComputerDialog
          trigger={
            <Button type="button" size="sm" className="ml-auto">
              <PlusIcon className="size-4" aria-hidden />
              Add a computer
            </Button>
          }
        />
      }
    >
      <div className="space-y-4">
        <HarnessReadinessLine />
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} />}
        {computers && computers.length === 0 && <EmptyRow flush>No computers yet</EmptyRow>}
        {computers && computers.length > 0 && (
          <ComputerOrderList computers={computers} />
        )}
        <OpenComputerCodes />
        <SharedComputersSection />
      </div>
    </SettingsCard>
  );
};
