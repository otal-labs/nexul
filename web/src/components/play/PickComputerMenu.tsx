import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { PickComputerGroup } from "@/components/play/PickComputerGroup";
import { signalActionClass } from "@/components/play/signalRowClass";
import { useFetchPresence, useListComputers } from "@/hooks/PairingHooks";
import { usePickQueuedComputer } from "@/hooks/PlayQueueHooks";
import type { PlayQueueItem } from "@/models/PlayQueue";
import { cn } from "@/lib/utils";

interface PickComputerMenuProps {
  item: PlayQueueItem;
}

// The run's person picks, or changes, the computer and T3 project it runs on; the run starts there and the project keeps it.
export const PickComputerMenu = ({ item }: PickComputerMenuProps) => {
  const { data: computers, isPending, error } = useListComputers();
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
        {computers && computers.length === 0 && <NoDataDisplay message="Add a computer in Settings to run plays." size="compact" />}
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
