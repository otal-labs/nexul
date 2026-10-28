import { closestCenter, DndContext, type DragEndEvent } from "@dnd-kit/core";
import { arrayMove, SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { useCallback, useMemo } from "react";

import { EnterList } from "@/components/EnterList";
import { ComputerOrderItem } from "@/components/settings/ComputerOrderItem";
import { settingsListClass } from "@/components/settings/SettingsRow";
import { useRowSortSensors } from "@/hooks/useRowSortSensors";
import { useFetchPresence, useReorderComputers } from "@/hooks/PairingHooks";
import type { Computer } from "@/models/Pairing";

interface ComputerOrderListProps {
  computers: Computer[];
}

// The person's computers ranked: runs go to the first that can take them, so the order is set by dragging a row.
export const ComputerOrderList = ({ computers }: ComputerOrderListProps) => {
  const presence = useFetchPresence();
  const { mutate: reorder } = useReorderComputers();
  const sensors = useRowSortSensors();
  const ids = useMemo(() => computers.map((c) => c.id), [computers]);

  const move = useCallback(
    (from: number, to: number) => {
      if (from === to || to < 0 || to >= ids.length) return;
      reorder(arrayMove(ids, from, to));
    },
    [ids, reorder],
  );
  const onDragEnd = ({ active, over }: DragEndEvent) => {
    if (over) move(ids.indexOf(String(active.id)), ids.indexOf(String(over.id)));
  };

  return (
    // dnd-kit mounts its screen-reader regions beside the list; the wrapper keeps them out of the card's spacing.
    <div>
      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
        <SortableContext items={ids} strategy={verticalListSortingStrategy}>
          <EnterList aria-label="Your computers, first runs first" className={settingsListClass}>
            {computers.map((computer, index) => (
              <ComputerOrderItem
                key={computer.id}
                computer={computer}
                index={index}
                count={computers.length}
                presence={presence.data?.[computer.id]}
                onMove={move}
              />
            ))}
          </EnterList>
        </SortableContext>
      </DndContext>
    </div>
  );
};
