import { DownloadIcon, ServerIcon } from "lucide-react";
import { Link } from "react-router";

import { EnterList } from "@/components/EnterList";
import { EmptyRow } from "@/components/EmptyRow";
import { AddRunnerDialog } from "@/components/runner/AddRunnerDialog";
import { EditableStackRoot } from "@/components/runner/EditableStackRoot";
import { RunnerRow } from "@/components/runner/RunnerRow";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Machine, Runner } from "@/models/Runner";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface MachineGroupProps {
  machine: Machine;
  runners: Runner[];
}

export const MachineGroup = ({ machine, runners }: MachineGroupProps) => {
  const canImport = useAreaAccess()?.("newProject") ?? false;
  const wsPath = useWorkspacePath();
  return (
    <section className="space-y-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <h2 className="flex items-start gap-2 text-sm font-semibold">
            <ServerIcon className="mt-0.5 size-4 shrink-0" aria-hidden />
            <span className="min-w-0 wrap-anywhere">{machine.name}</span>
          </h2>
          <p className="flex flex-wrap items-center gap-x-2 gap-y-1 font-mono text-xs text-muted-foreground">
            {machine.reported_hostname && machine.reported_hostname !== machine.name && (
              <span className="wrap-anywhere">{machine.reported_hostname}</span>
            )}
            <span>
              {runners.length} runner{runners.length === 1 ? "" : "s"}
            </span>
            <span>last seen {formatRelativeTime(machine.last_seen)}</span>
            <span>root <EditableStackRoot machineId={machine.id} stackRoot={machine.stack_root} /></span>
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {canImport && (
            <Button variant="outline" size="sm" asChild>
              <Link to={wsPath(`/wizard/project/import?machine=${machine.id}`)}>
                <DownloadIcon className="size-3.5" /> Import from this machine
              </Link>
            </Button>
          )}
          <AddRunnerDialog machineName={machine.name} triggerSize="sm" triggerVariant="outline" />
        </div>
      </div>
      {runners.length === 0 && <EmptyRow flush>No runners on this machine yet.</EmptyRow>}
      {runners.length > 0 && (
        <EnterList className="divide-y divide-border rounded-lg border border-border bg-card">
          {runners.map((runner) => (
            <RunnerRow key={runner.id} runner={runner} />
          ))}
        </EnterList>
      )}
    </section>
  );
};
