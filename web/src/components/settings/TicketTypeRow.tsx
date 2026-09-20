import { useState, type FocusEvent } from "react";

import { ColorSwatchButton } from "@/components/settings/ColorSwatchButton";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { useDeleteTicketType, useRenameTicketType } from "@/hooks/TicketTypeHooks";
import type { TicketType } from "@/models/TicketType";

interface TicketTypeRowProps {
  type: TicketType;
}

export const TicketTypeRow = ({ type }: TicketTypeRowProps) => {
  const renameTicketType = useRenameTicketType();
  const deleteTicketType = useDeleteTicketType();
  const [editing, setEditing] = useState(false);

  // Full-replacement PATCH — resend the untouched field so it isn't silently cleared.
  const commitRename = (event: FocusEvent<HTMLInputElement>) => {
    const name = event.target.value.trim();
    if (name && name !== type.name) {
      void renameTicketType.mutateAsync({ id: type.id, name, color: type.color });
    }
    setEditing(false);
  };

  return (
    <li className="-mx-2 flex items-center gap-2 rounded-sm px-2 py-1.5 text-sm transition-colors duration-[120ms] ease-standard hover:bg-accent/40">
      {editing && (
        <input
          className="flex-1 rounded-md border border-input px-2 py-1 text-sm"
          aria-label="Ticket type name"
          defaultValue={type.name}
          autoFocus
          onBlur={commitRename}
          onKeyDown={(event) => {
            if (event.key === "Enter") (event.target as HTMLInputElement).blur();
            if (event.key === "Escape") setEditing(false);
          }}
        />
      )}
      {!editing && <span className="flex-1 text-sm font-medium">{type.name}</span>}
      <ColorSwatchButton
        label={`Color for ${type.name}`}
        value={type.color}
        onChange={(color) => void renameTicketType.mutateAsync({ id: type.id, name: type.name, color })}
      />
      <RowActionsMenu
        subject={type.name}
        actions={[
          { label: "Rename", onSelect: () => setEditing(true) },
          { label: "Delete", destructive: true, onSelect: () => deleteTicketType.mutate(type.id) },
        ]}
      />
    </li>
  );
};
