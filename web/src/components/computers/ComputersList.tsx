import { closestCenter, DndContext, type DragEndEvent } from "@dnd-kit/core";
import { arrayMove, SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";

import { ComputerListRow } from "@/components/computers/ComputerListRow";
import { RunSettingsGroup } from "@/components/computers/RunSettingsGroup";
import { UnfinishedSetupsRow } from "@/components/computers/UnfinishedSetupsRow";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ListPaneGroup } from "@/components/listpane/ListPaneGroup";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { OpenComputerCodes } from "@/components/settings/OpenComputerCodes";
import { useListSharedComputers } from "@/hooks/ComputerHooks";
import { useListOrderComputers, useReorderComputers } from "@/hooks/PairingHooks";
import { useRowSortSensors } from "@/hooks/useRowSortSensors";
import { sharedOutOfOrder } from "@/models/SharedComputer";

interface ComputersListProps {
  current: string;
}

// Your computers in run order (ADR 0165) with those shared with you beside them, then where runs go.
export const ComputersList = ({ current }: ComputersListProps) => {
  const { data: computers, isPending, error } = useListOrderComputers();
  const { data: sharedAll } = useListSharedComputers();
  const { mutate: reorder } = useReorderComputers();
  const sensors = useRowSortSensors();
  const ids = computers?.map((c) => c.id) ?? [];
  const shared = sharedOutOfOrder(sharedAll ?? []);

  const onDragEnd = ({ active, over }: DragEndEvent) => {
    const from = ids.indexOf(String(active.id));
    const to = over ? ids.indexOf(String(over.id)) : -1;
    if (from < 0 || to < 0 || from === to) return;
    reorder(arrayMove(ids, from, to));
  };

  return (
    <div className="pb-8">
      {isPending && <LoadingDisplay label="Loading computers…" />}
      {error && <ErrorDisplay error={error} title="Couldn't load your computers." />}
      {computers && computers.length > 0 && (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
          <SortableContext items={ids} strategy={verticalListSortingStrategy}>
            <ListPaneGroup plain label="Your computers" caption="Your computers · run order" count={computers.length}>
              {computers.map((computer) => (
                <ComputerListRow key={computer.id} computer={computer} rank={ids.indexOf(computer.id) + 1} sortable={ids.length > 1} selected={computer.id === current} />
              ))}
            </ListPaneGroup>
          </SortableContext>
        </DndContext>
      )}
      {computers && computers.length === 0 && (
        <ListPaneGroup plain label="Your computers" count={0}>
          <li>
            <EmptyRow className="px-3 py-2 text-xs">No computers yet</EmptyRow>
          </li>
        </ListPaneGroup>
      )}
      {shared.length > 0 && (
        <ListPaneGroup plain label="Shared with you" count={shared.length}>
          {shared.map((computer) => (
            <ComputerListRow key={computer.id} computer={computer} sortable={false} selected={computer.id === current} />
          ))}
        </ListPaneGroup>
      )}
      <UnfinishedSetupsRow />
      <div className="px-3 empty:hidden">
        <OpenComputerCodes />
      </div>
      <RunSettingsGroup current={current} />
    </div>
  );
};
