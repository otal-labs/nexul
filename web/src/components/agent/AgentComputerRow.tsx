import { useSortable } from "@dnd-kit/sortable";
import { UsersIcon } from "lucide-react";
import { useReducedMotion } from "motion/react";

import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { statusDotClass } from "@/components/settings/SettingsStatus";
import { useFetchPresence } from "@/hooks/PairingHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { EASE_OUT } from "@/lib/motion";
import { cn } from "@/lib/utils";
import { agentPath } from "@/models/Agent";
import { computerLanes, runnerPresence, worstFirst } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";

interface AgentComputerRowProps {
  computer: Computer;
  rank: number;
  sortable: boolean;
  selected: boolean;
}

// One computer in your run order, yours or shared: its state as a dot, its name and place; the row drags to a new place.
export const AgentComputerRow = ({ computer, rank, sortable, selected }: AgentComputerRowProps) => {
  const wsPath = useWorkspacePath();
  const presence = useFetchPresence();
  const reduced = useReducedMotion() ?? false;
  const { setNodeRef, listeners, transform, transition, isDragging } = useSortable({
    id: computer.id,
    disabled: !sortable,
    transition: reduced ? null : { duration: 200, easing: EASE_OUT },
  });
  // A shared computer shows its owner's name and only whether it is online, never its owner's setup.
  const [lane] = computer.shared ? [runnerPresence(computer.runner)] : worstFirst(computerLanes(computer, presence.data?.[computer.id]));
  const name = computer.shared ? `${computer.shared.owner_name}'s ${computer.name}` : computer.name || "New computer";

  return (
    <ListPaneRow
      to={wsPath(agentPath("computers", computer.id))}
      title={name}
      muted={!computer.name || !!computer.plays_off}
      titleIcon={computer.shared && <UsersIcon role="img" aria-label="Shared with you" className="size-3 text-muted-foreground" />}
      leading={lane && <span aria-label={lane.text} role="img" className={cn("size-1.5 rounded-full", statusDotClass[lane.tone])} />}
      meta={<span className="font-mono text-micro text-muted-foreground tabular-nums">#{rank}</span>}
      dense
      selected={selected}
      {...(sortable && {
        drag: {
          setNodeRef,
          listeners,
          isDragging,
          style: { transform: transform ? `translate3d(0, ${Math.round(transform.y)}px, 0)` : undefined, transition },
        },
      })}
    />
  );
};
