import { useSortable } from "@dnd-kit/sortable";
import { UsersIcon } from "lucide-react";
import { useReducedMotion } from "motion/react";

import { DrillChevron } from "@/components/computers/DrillChevron";
import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { SetupProviderMarks } from "@/components/settings/SetupProviderMarks";
import { statusDotFor } from "@/components/settings/SettingsStatus";
import { useFetchComputerSetup } from "@/hooks/ComputerSetupHooks";
import { useFetchPresence } from "@/hooks/PairingHooks";
import { EASE_OUT } from "@/lib/motion";
import { cn } from "@/lib/utils";
import { computerModelName } from "@/models/ModelPick";
import { computersSearch } from "@/models/Agent";
import { computerStateLine } from "@/models/ComputerChecks";
import { providerSetupLines, stillPairing, type Computer } from "@/models/Pairing";

interface ComputerListRowProps {
  computer: Computer;
  rank?: number;
  sortable: boolean;
  selected: boolean;
}

// One computer in your run order, yours or shared: its state as a dot, its name and place, one short state line and its
// set-up providers as marks; the row drags to a new place.
export const ComputerListRow = ({ computer, rank, sortable, selected }: ComputerListRowProps) => {
  const presence = useFetchPresence();
  const reduced = useReducedMotion() ?? false;
  const own = !computer.shared && !!computer.runner && !stillPairing(computer);
  const { data: setup } = useFetchComputerSetup(own ? computer.id : "");
  const { setNodeRef, listeners, transform, transition, isDragging } = useSortable({
    id: computer.id,
    disabled: !sortable,
    transition: reduced ? null : { duration: 200, easing: EASE_OUT },
  });
  const state = computerStateLine(computer, presence.data?.[computer.id]);
  const confirmed = setup ? providerSetupLines(setup).filter((l) => l.state === "confirmed") : [];
  const name = computer.shared ? `${computer.shared.owner_name}'s ${computer.name}` : computer.name || "New computer";

  return (
    <ListPaneRow
      to={computersSearch(computer.id)}
      title={name}
      muted={!computer.name || !!computer.plays_off}
      titleIcon={computer.shared && <UsersIcon role="img" aria-label="Shared with you" className="size-3 text-muted-foreground" />}
      leading={<span aria-hidden className={cn("size-1.5 rounded-full", statusDotFor(state.tone, state.hollow))} />}
      meta={
        <span className="flex items-center gap-1.5">
          {rank && (
            <span title="Run order" className="font-mono text-micro text-muted-foreground tabular-nums">
              #{rank}
            </span>
          )}
          {confirmed.length > 0 && (
            <SetupProviderMarks lines={confirmed} titleOf={(line) => (setup ? `${line.name} · ${computerModelName(computer, setup, line.provider)}` : line.name)} />
          )}
        </span>
      }
      metaOnTitleLine
      trailing={<DrillChevron />}
      detail={<span className={cn("min-w-0 truncate", state.tone === "warning" && "text-warning")}>{state.text}</span>}
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
