import {
  closestCenter,
  DndContext,
  KeyboardSensor,
  MouseSensor,
  TouchSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import { arrayMove, SortableContext, sortableKeyboardCoordinates, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { useCallback, useMemo } from "react";

import { EnterList } from "@/components/EnterList";
import { ComputerOrderItem } from "@/components/settings/ComputerOrderItem";
import { useFetchPresence, useReorderComputers } from "@/hooks/PairingHooks";
import type { Computer } from "@/models/Pairing";

const MOUSE = { activationConstraint: { distance: 2 } };
const TOUCH = { activationConstraint: { delay: 200, tolerance: 8 } };
const KEYS = { coordinateGetter: sortableKeyboardCoordinates };

interface ComputerOrderListProps {
  computers: Computer[];
}

// The person's computers ranked: runs go to the first that can take them, so the order is set by dragging a row.
export const ComputerOrderList = ({ computers }: ComputerOrderListProps) => {
  const presence = useFetchPresence();
  const { mutate: reorder } = useReorderComputers();
  const sensors = useSensors(useSensor(MouseSensor, MOUSE), useSensor(TouchSensor, TOUCH), useSensor(KeyboardSensor, KEYS));
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
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
      <SortableContext items={ids} strategy={verticalListSortingStrategy}>
        <EnterList aria-label="Your computers, first runs first" className="ml-10 divide-y divide-border rounded-md border">
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
  );
};
