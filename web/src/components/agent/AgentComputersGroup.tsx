import { closestCenter, DndContext, type DragEndEvent } from "@dnd-kit/core";
import { arrayMove, SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { useParams } from "react-router";

import { AgentComputerRow } from "@/components/agent/AgentComputerRow";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ListPaneGroup } from "@/components/listpane/ListPaneGroup";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { OpenComputerCodes } from "@/components/settings/OpenComputerCodes";
import { useListComputers, useReorderComputers } from "@/hooks/PairingHooks";
import { useRowSortSensors } from "@/hooks/useRowSortSensors";

interface AgentComputersGroupProps {
  query: string;
}

// Your computers in run order (ADR 0165); a filtered list can't say where a dropped row lands, so a search stops dragging.
export const AgentComputersGroup = ({ query }: AgentComputersGroupProps) => {
  const { kind, id } = useParams();
  const { data: computers, isPending, error } = useListComputers();
  const { mutate: reorder } = useReorderComputers();
  const sensors = useRowSortSensors();
  const ids = computers?.map((c) => c.id) ?? [];
  const shown = computers?.filter((c) => query === "" || c.name.toLowerCase().includes(query));

  const onDragEnd = ({ active, over }: DragEndEvent) => {
    const from = ids.indexOf(String(active.id));
    const to = over ? ids.indexOf(String(over.id)) : -1;
    if (from < 0 || to < 0 || from === to) return;
    reorder(arrayMove(ids, from, to));
  };

  return (
    <div>
      {isPending && <LoadingDisplay label="Loading computers…" />}
      {error && <ErrorDisplay error={error} title="Couldn't load your computers." />}
      {shown && shown.length > 0 && (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
          <SortableContext items={ids} strategy={verticalListSortingStrategy}>
            <ListPaneGroup label="Your computers" count={shown.length}>
              {shown.map((computer) => (
                <AgentComputerRow
                  key={computer.id}
                  computer={computer}
                  rank={ids.indexOf(computer.id) + 1}
                  sortable={query === "" && ids.length > 1}
                  selected={kind === "computers" && computer.id === id}
                />
              ))}
            </ListPaneGroup>
          </SortableContext>
        </DndContext>
      )}
      {computers && computers.length === 0 && query === "" && (
        <ListPaneGroup label="Your computers" count={0}>
          <li>
            <EmptyRow className="px-3 py-2 text-xs">No computers yet</EmptyRow>
          </li>
        </ListPaneGroup>
      )}
      <div className="px-3 empty:hidden">
        <OpenComputerCodes />
      </div>
    </div>
  );
};
