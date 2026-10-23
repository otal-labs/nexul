import { closestCenter, DndContext, type DragEndEvent } from "@dnd-kit/core";
import { arrayMove, SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { Fragment, type ReactNode } from "react";

import { AddComputerRow } from "@/components/settings/AddComputerRow";
import type { RowControls } from "@/components/settings/RankedComputerRow";
import { RunSettingField } from "@/components/settings/RunSettingField";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useRowSortSensors } from "@/hooks/useRowSortSensors";
import { cn } from "@/lib/utils";
import type { Computer } from "@/models/Pairing";

const YOUR_ORDER = "yours";
const CUSTOM = "custom";

interface RankedComputersFieldProps {
  // What it ranks computers for, as "this <subject>" reads in its labels.
  subject: string;
  // The person's own computers, in their order.
  own: Computer[];
  // Empty follows the person's order; otherwise the computers in the order a run tries them.
  ids: string[];
  onChange: (ids: string[]) => void;
  disabled?: boolean;
  // Computers listed in the Add menu that cannot be picked.
  unavailable?: { id: string; label: string }[];
  // Puts the segmented Your order / Custom control in a labelled settings field; a field whose owner picks the mode elsewhere leaves it out.
  label?: string;
  toggle?: boolean;
  renderRow: (row: { id: string; rank: number; controls: RowControls }) => ReactNode;
}

// The ordered picker of a project and of an auto play: a ranked list that is dragged into order, with an Add row at its end.
export const RankedComputersField = ({
  subject,
  own,
  ids,
  onChange,
  disabled = false,
  unavailable = [],
  label,
  toggle = true,
  renderRow,
}: RankedComputersFieldProps) => {
  const sensors = useRowSortSensors();
  const custom = ids.length > 0;
  const editable = custom && !disabled;
  const listed = custom ? ids : own.map((c) => c.id);
  const rest = own.filter((c) => !ids.includes(c.id));
  const controlsFor = (id: string): RowControls => ({
    sortable: editable && ids.length > 1,
    onRemove: editable && ids.length > 1 ? () => onChange(ids.filter((x) => x !== id)) : undefined,
  });
  const onDragEnd = ({ active, over }: DragEndEvent) => {
    if (over && active.id !== over.id) onChange(arrayMove(ids, ids.indexOf(String(active.id)), ids.indexOf(String(over.id))));
  };

  const modes = (
    <ToggleGroup
      type="single"
      variant="segmented"
      size="xs"
      value={custom ? CUSTOM : YOUR_ORDER}
      // Radix reports "" when the chosen item is clicked again; keep the current choice.
      onValueChange={(v) => v && onChange(v === CUSTOM ? own.map((c) => c.id) : [])}
      disabled={disabled}
      aria-label={`Computers for this ${subject}`}
      className="justify-self-start"
    >
      <ToggleGroupItem value={YOUR_ORDER}>Your order</ToggleGroupItem>
      <ToggleGroupItem value={CUSTOM}>Custom</ToggleGroupItem>
    </ToggleGroup>
  );

  return (
    <div>
      {toggle && label && <RunSettingField label={label}>{modes}</RunSettingField>}
      {toggle && !label && modes}
      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
        <SortableContext items={listed} strategy={verticalListSortingStrategy}>
          <ul aria-label={`This ${subject}'s computers`} className={cn("divide-y divide-border rounded-md border border-border bg-card", toggle && "mt-1")}>
            {listed.map((id, index) => (
              <Fragment key={id}>{renderRow({ id, rank: index + 1, controls: controlsFor(id) })}</Fragment>
            ))}
            {editable && rest.length > 0 && <AddComputerRow computers={rest} unavailable={unavailable} onAdd={(id) => onChange([...ids, id])} />}
          </ul>
        </SortableContext>
      </DndContext>
    </div>
  );
};
