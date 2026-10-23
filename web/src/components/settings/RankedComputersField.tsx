import { arrayMove } from "@dnd-kit/sortable";
import { Fragment, type ReactNode } from "react";

import { RunSettingField } from "@/components/settings/RunSettingField";
import type { RowAction } from "@/components/settings/RowActionsMenu";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
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
  // False leaves both segments unselected and the list hidden until one is chosen.
  chosen?: boolean;
  disabled?: boolean;
  // Computers listed in the Add menu that cannot be picked.
  unavailable?: { id: string; label: string }[];
  // Puts the segmented control in a labelled settings field.
  label?: string;
  renderRow: (row: { id: string; rank: number; actions: RowAction[] }) => ReactNode;
}

// The ordered picker of a project and of an auto play: Your order or Custom, over a ranked list and an Add menu.
export const RankedComputersField = ({
  subject,
  own,
  ids,
  onChange,
  chosen = true,
  disabled = false,
  unavailable = [],
  label,
  renderRow,
}: RankedComputersFieldProps) => {
  const custom = ids.length > 0;
  const listed = custom ? ids : own.map((c) => c.id);
  const rest = own.filter((c) => !ids.includes(c.id));
  const actionsFor = (id: string, index: number): RowAction[] => {
    if (!custom || disabled) return [];
    return [
      { label: "Move up", disabled: index === 0, onSelect: () => onChange(arrayMove(ids, index, index - 1)) },
      { label: "Move down", disabled: index === ids.length - 1, onSelect: () => onChange(arrayMove(ids, index, index + 1)) },
      { label: `Remove from this ${subject}`, disabled: ids.length === 1, onSelect: () => onChange(ids.filter((x) => x !== id)) },
    ];
  };

  const toggle = (
    <ToggleGroup
      type="single"
      variant="segmented"
      size="xs"
      value={chosen ? (custom ? CUSTOM : YOUR_ORDER) : ""}
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
      {label && <RunSettingField label={label}>{toggle}</RunSettingField>}
      {!label && toggle}
      {chosen && (
        <ul aria-label={`This ${subject}'s computers`} className="mt-1 divide-y divide-border rounded-md border border-border bg-card">
          {listed.map((id, index) => (
            <Fragment key={id}>{renderRow({ id, rank: index + 1, actions: actionsFor(id, index) })}</Fragment>
          ))}
        </ul>
      )}
      {chosen && custom && rest.length > 0 && (
        <Select value="" onValueChange={(id) => onChange([...ids, id])} disabled={disabled}>
          <SelectTrigger aria-label={`Add a computer to this ${subject}`} className="mt-2 h-8 w-52 text-xs">
            <SelectValue placeholder="Add a computer" />
          </SelectTrigger>
          <SelectContent>
            {rest.map((c) => (
              <SelectItem key={c.id} value={c.id} className="text-xs">
                {c.name}
              </SelectItem>
            ))}
            {unavailable.map((c) => (
              <SelectItem key={c.id} value={c.id} disabled className="text-xs">
                {c.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
    </div>
  );
};
