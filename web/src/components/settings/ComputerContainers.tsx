import { useState } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ComputerFact } from "@/components/settings/ComputerFact";
import { ComputerContainerLogsSheet } from "@/components/settings/ComputerContainerLogsSheet";
import { ComputerContainerRow } from "@/components/settings/ComputerContainerRow";
import { useFetchComputerContainers } from "@/hooks/ComputerContainerHooks";

interface ComputerContainersProps {
  computerId: string;
  dockerVersion: string | undefined;
}

// The computer's own Docker containers, each opening its logs; they are not Nexul services and only the owner sees them.
export const ComputerContainers = ({ computerId, dockerVersion }: ComputerContainersProps) => {
  const { data: containers, isPending, error } = useFetchComputerContainers(computerId);
  const [shown, setShown] = useState({ name: "", open: false });

  return (
    <ComputerFact label="Containers">
      <div className="space-y-1">
        {dockerVersion && <p className="font-mono text-muted-foreground">Docker {dockerVersion}</p>}
        {isPending && <LoadingDisplay label="Loading containers" className="justify-start p-0" />}
        {error && <ErrorDisplay error={error} title="Couldn't list the containers." className="p-3" />}
        {containers && containers.length === 0 && <EmptyRow flush className="text-xs">No containers on this computer.</EmptyRow>}
        {containers && containers.length > 0 && (
          <ul className="divide-y divide-border">
            {containers.map((container) => (
              <ComputerContainerRow key={container.name} container={container} onLogs={(name) => setShown({ name, open: true })} />
            ))}
          </ul>
        )}
      </div>
      {shown.name && (
        <ComputerContainerLogsSheet
          computerId={computerId}
          container={shown.name}
          open={shown.open}
          onOpenChange={(open) => setShown((prev) => ({ ...prev, open }))}
        />
      )}
    </ComputerFact>
  );
};
