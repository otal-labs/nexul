import { useState, type FocusEvent, type KeyboardEvent } from "react";

import { isStatusIconName, StatusIcon } from "@/components/board/StatusIcon";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { StatusIconPicker } from "@/components/settings/StatusIconPicker";
import { useDeleteStatus, useRenameStatus } from "@/hooks/StatusHooks";
import { cn } from "@/lib/utils";
import { statusStage, type BoardStatus } from "@/models/Status";

interface StatusRowProps {
  status: BoardStatus;
  index: number;
  total: number;
  onMove: (direction: -1 | 1) => void;
}

export const StatusRow = ({ status, index, total, onMove }: StatusRowProps) => {
  const renameStatus = useRenameStatus();
  const deleteStatus = useDeleteStatus();
  const [editing, setEditing] = useState(false);
  const [nameDraft, setNameDraft] = useState(status.name);
  const [iconDraft, setIconDraft] = useState(status.icon);

  const startEditing = () => {
    setNameDraft(status.name);
    setIconDraft(status.icon);
    setEditing(true);
  };

  // Full-replacement call so untouched fields aren't cleared; one atomic commit path.
  const commit = () => {
    const name = nameDraft.trim();
    if (name && (name !== status.name || iconDraft !== status.icon)) {
      void renameStatus.mutateAsync({ id: status.id, name, kind: status.kind, icon: iconDraft });
    }
    setEditing(false);
  };

  const handleGroupBlur = (event: FocusEvent<HTMLDivElement>) => {
    if (!event.currentTarget.contains(event.relatedTarget as Node | null)) commit();
  };

  const handleNameKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") commit();
    if (event.key === "Escape") setEditing(false);
  };

  const stage = statusStage(status.kind);
  const kindDotClass = cn("size-1.5 shrink-0 rounded-full", stage.dot);
  const kindIconClass = cn("size-3.5 shrink-0", stage.text);

  return (
    <li className="-mx-2 flex items-center gap-2 rounded-sm px-2 py-1.5 text-sm transition-colors duration-[120ms] ease-standard hover:bg-accent/40">
      {editing && (
        <div onBlur={handleGroupBlur} className="flex flex-1 flex-wrap items-center gap-3">
          <input
            className="min-w-0 flex-1 rounded-md border border-input px-2 py-1 text-sm"
            aria-label="Status name"
            value={nameDraft}
            autoFocus
            onChange={(event) => setNameDraft(event.target.value)}
            onKeyDown={handleNameKeyDown}
          />
          <StatusIconPicker label="Icon" value={iconDraft} onChange={setIconDraft} />
        </div>
      )}
      {!editing && (
        <span className="flex flex-1 items-center gap-2 text-sm font-medium">
          {isStatusIconName(status.icon) ? (
            <StatusIcon icon={status.icon} className={kindIconClass} />
          ) : (
            <span className={cn(kindDotClass, "mx-1")} aria-hidden />
          )}
          {status.name}
        </span>
      )}
      <RowActionsMenu
        subject={status.name}
        actions={[
          { label: "Rename", onSelect: startEditing },
          { label: "Move up", disabled: index === 0, onSelect: () => onMove(-1) },
          { label: "Move down", disabled: index === total - 1, onSelect: () => onMove(1) },
          { label: "Delete", destructive: true, onSelect: () => deleteStatus.mutate(status.id) },
        ]}
      />
    </li>
  );
};
