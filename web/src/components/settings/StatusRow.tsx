import { useState, type FocusEvent, type KeyboardEvent } from "react";

import { isStatusIconName, StatusIcon } from "@/components/board/StatusIcon";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { SettingsListRow } from "@/components/settings/SettingsListRow";
import { StatusIconPicker } from "@/components/settings/StatusIconPicker";
import { useFetchProjectTicketCounts } from "@/hooks/ProjectHooks";
import { useDeleteStatus, useRenameStatus } from "@/hooks/StatusHooks";
import { cn } from "@/lib/utils";
import { statusStage, type BoardStatus } from "@/models/Status";
import { countLabel } from "@/utils/CountUtility";

interface StatusRowProps {
  projectId: string;
  status: BoardStatus;
  index: number;
  total: number;
  onMove: (direction: -1 | 1) => void;
}

export const StatusRow = ({ projectId, status, index, total, onMove }: StatusRowProps) => {
  const renameStatus = useRenameStatus();
  const deleteStatus = useDeleteStatus();
  const [editing, setEditing] = useState(false);
  const [nameDraft, setNameDraft] = useState(status.name);
  const [iconDraft, setIconDraft] = useState(status.icon);
  const { data: counts } = useFetchProjectTicketCounts(projectId);
  const held = counts?.tickets_by_status[status.id] ?? 0;

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
  const actions = (
    <RowActionsMenu
      subject={status.name}
      projectId={projectId}
      actions={[
        { label: "Rename", permission: "projects:write", verb: "rename statuses", onSelect: startEditing },
        { label: "Move up", permission: "projects:write", verb: "reorder statuses", disabled: index === 0, onSelect: () => onMove(-1) },
        { label: "Move down", permission: "projects:write", verb: "reorder statuses", disabled: index === total - 1, onSelect: () => onMove(1) },
        {
          label: "Delete",
          destructive: true,
          permission: "projects:delete",
          verb: "delete statuses",
          // The server keeps a column that still holds tickets, so the menu says so instead of failing on click.
          disabled: held > 0,
          hint: held > 0 ? `Used by ${countLabel(held, "ticket")}` : undefined,
          onSelect: () => deleteStatus.mutate(status.id),
        },
      ]}
    />
  );

  return (
    <>
      {editing && (
        <li className="flex min-h-11 items-center">
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
        </li>
      )}
      {!editing && (
        <SettingsListRow
          lead={
            <>
              {isStatusIconName(status.icon) && <StatusIcon icon={status.icon} className={cn("size-3.5", stage.text)} />}
              {!isStatusIconName(status.icon) && <span className={cn("size-2 rounded-full", stage.dot)} aria-hidden />}
            </>
          }
          name={<span className="truncate font-medium" title={status.name}>{status.name}</span>}
          count={counts && countLabel(held, "ticket")}
          actions={actions}
        />
      )}
    </>
  );
};
