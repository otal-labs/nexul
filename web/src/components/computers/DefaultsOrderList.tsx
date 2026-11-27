import { closestCenter, DndContext, type DragEndEvent } from "@dnd-kit/core";
import { arrayMove, SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";

import { RankedComputerRow } from "@/components/settings/RankedComputerRow";
import { useListOrderComputers, useReorderComputers } from "@/hooks/PairingHooks";
import { useRowSortSensors } from "@/hooks/useRowSortSensors";
import { sharedComputerName } from "@/models/SharedComputer";

// Your order as the Defaults use it: the same ranked list a project's own order is, reordering the one list the dialog's left column shows.
export const DefaultsOrderList = () => {
  const { data: computers } = useListOrderComputers();
  const { mutate: reorder } = useReorderComputers();
  const sensors = useRowSortSensors();
  if (!computers || computers.length === 0) return null;

  const ids = computers.map((c) => c.id);
  const onDragEnd = ({ active, over }: DragEndEvent) => {
    const from = ids.indexOf(String(active.id));
    const to = over ? ids.indexOf(String(over.id)) : -1;
    if (from < 0 || to < 0 || from === to) return;
    reorder(arrayMove(ids, from, to));
  };

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
      <SortableContext items={ids} strategy={verticalListSortingStrategy}>
        <ul aria-label="Your order" className="divide-y divide-border rounded-md border border-border bg-card">
          {computers.map((computer, i) => (
            <RankedComputerRow key={computer.id} id={computer.id} name={sharedComputerName(computer)} computer={computer} rank={i + 1} controls={{ sortable: ids.length > 1 }} />
          ))}
        </ul>
      </SortableContext>
    </DndContext>
  );
};
