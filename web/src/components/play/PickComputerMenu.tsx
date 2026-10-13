import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoComputersDisplay } from "@/components/play/NoComputersDisplay";
import { PickComputerGroup } from "@/components/play/PickComputerGroup";
import { signalActionClass } from "@/components/play/signalRowClass";
import { useListSharedComputers } from "@/hooks/ComputerHooks";
import { useFetchPresence, useListComputers } from "@/hooks/PairingHooks";
import { usePickQueuedComputer } from "@/hooks/PlayQueueHooks";
import type { PlayQueueItem } from "@/models/PlayQueue";
import { notRunnableYet, sharedComputerName } from "@/models/SharedComputer";
import { cn } from "@/lib/utils";

interface PickComputerMenuProps {
  item: PlayQueueItem;
}

// The run's person picks, or changes, the computer and T3 project it runs on; the run starts there and the project keeps it.
export const PickComputerMenu = ({ item }: PickComputerMenuProps) => {
  const { data: computers, isPending, error } = useListComputers();
  const { data: shared } = useListSharedComputers();
  const { data: presence } = useFetchPresence();
  const pick = usePickQueuedComputer();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className={cn(signalActionClass, "-ml-1.5 mt-1")} loading={pick.isPending}>
          {item.computer_id === "" ? "Pick a computer" : "Change computer"}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-64">
        {isPending && <LoadingDisplay label="Loading computers" className="justify-start p-2" />}
        {error && <ErrorDisplay error={error} title="Couldn't load your computers." className="p-3" />}
        {computers && computers.length === 0 && <NoComputersDisplay />}
        {shared?.map((computer) => (
          <DropdownMenuItem key={computer.id} disabled>
            <span className="min-w-0 flex-1 truncate">{sharedComputerName(computer)}</span>
            <span className="text-xs text-muted-foreground">{notRunnableYet}</span>
          </DropdownMenuItem>
        ))}
        {computers?.map((computer, index) => (
          <PickComputerGroup
            key={computer.id}
            computer={computer}
            connected={presence?.[computer.id] === "connected"}
            separated={index > 0}
            onPick={(computerId, harnessProjectId) => pick.mutate({ itemId: item.id, computerId, harnessProjectId })}
          />
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
