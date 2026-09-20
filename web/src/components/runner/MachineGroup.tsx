import { DownloadIcon, ServerIcon } from "lucide-react";
import { Link } from "react-router";

import { AddRunnerDialog } from "@/components/runner/AddRunnerDialog";
import { EditableStackRoot } from "@/components/runner/EditableStackRoot";
import { RunnerRow } from "@/components/runner/RunnerRow";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { Button } from "@/components/ui/button";
import type { Machine, Runner } from "@/models/Runner";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface MachineGroupProps {
  machine: Machine;
  runners: Runner[];
}

export const MachineGroup = ({ machine, runners }: MachineGroupProps) => (
  <section className="space-y-3">
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div className="min-w-0 space-y-1">
        <h2 className="flex items-center gap-2 text-sm font-semibold">
          <ServerIcon className="size-4 shrink-0" aria-hidden />
          {machine.name}
        </h2>
        <p className="flex flex-wrap items-center gap-x-2 gap-y-1 font-mono text-xs text-muted-foreground">
          {machine.reported_hostname && machine.reported_hostname !== machine.name && <span>{machine.reported_hostname}</span>}
          <span>
            {runners.length} runner{runners.length === 1 ? "" : "s"}
          </span>
          <span>last seen {formatRelativeTime(machine.last_seen)}</span>
          <span>root <EditableStackRoot machineId={machine.id} stackRoot={machine.stack_root} /></span>
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="sm" asChild>
          <Link to={`/wizard/project/import?machine=${machine.id}`}>
            <DownloadIcon className="size-3.5" /> Import from this machine
          </Link>
        </Button>
        <AddRunnerDialog machineName={machine.name} triggerSize="sm" triggerVariant="outline" />
      </div>
    </div>
    {runners.length === 0 && <NoDataDisplay size="compact" message="No runners on this machine yet." />}
    {runners.length > 0 && (
      <ul className="divide-y divide-border rounded-lg border border-border bg-card">
        {runners.map((runner, index) => (
          <RunnerRow key={runner.id} runner={runner} index={index} />
        ))}
      </ul>
    )}
  </section>
);
