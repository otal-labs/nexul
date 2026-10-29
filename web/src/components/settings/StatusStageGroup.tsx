import { PlusIcon } from "lucide-react";

import { EnterList } from "@/components/EnterList";
import { EmptyRow } from "@/components/EmptyRow";
import { Microheader } from "@/components/Microheader";
import { NewStatusForm } from "@/components/settings/NewStatusForm";
import { StatusRow } from "@/components/settings/StatusRow";
import { settingsRowListClass } from "@/components/settings/SettingsRow";
import { ActionButton } from "@/components/action/ActionButton";
import type { BoardStatus, StatusKind } from "@/models/Status";

interface StatusStageGroupProps {
  projectId: string;
  kind: StatusKind;
  label: string;
  // Each row keeps its index in the project's full order, which is what a move swaps.
  rows: { status: BoardStatus; index: number }[];
  total: number;
  adding: boolean;
  onToggleAdding: () => void;
  onMove: (index: number, direction: -1 | 1) => void;
}

export const StatusStageGroup = ({ projectId, kind, label, rows, total, adding, onToggleAdding, onMove }: StatusStageGroupProps) => (
  <div>
    <div className="flex h-8 items-center justify-between gap-2">
      <Microheader>{label}</Microheader>
      <ActionButton permission="projects:write" projectId={projectId} verb="add a status" variant="ghost" size="sm" icon={PlusIcon} className="w-8 px-0" aria-label={`Add ${kind} status`} onClick={onToggleAdding} />
    </div>
    {rows.length === 0 && (
      <EmptyRow className="-mx-2 flex min-h-11 items-center border-t border-border px-2 py-0">No {kind} statuses, so the board skips this stage.</EmptyRow>
    )}
    {rows.length > 0 && (
      // Named for assistive tech since the header states kind once and rows don't repeat it.
      <EnterList aria-label={`${label} statuses`} className={settingsRowListClass}>
        {rows.map(({ status, index }) => (
          <StatusRow key={status.id} projectId={projectId} status={status} index={index} total={total} onMove={(direction) => onMove(index, direction)} />
        ))}
      </EnterList>
    )}
    {adding && <NewStatusForm projectId={projectId} kind={kind} onDone={onToggleAdding} />}
  </div>
);
