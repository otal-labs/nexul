import { cn } from "@/lib/utils";
import { managerLabel, type AutomationHostManager, type ManagerState } from "@/models/AutomationHost";

interface AutomationHostManagersProps {
  managers: readonly AutomationHostManager[];
}

const stateDot: Record<ManagerState, string> = {
  ready: "bg-success",
  "token rejected": "bg-destructive",
  unreachable: "bg-warning-mark",
};

// One line per workspace token the host holds: the workspace, then the manager's name and, when its token does not work, why.
export const AutomationHostManagers = ({ managers }: AutomationHostManagersProps) =>
  managers.length > 0 && (
    <ul aria-label="Secrets managers" className="mt-1 flex flex-wrap gap-x-4 gap-y-0.5 text-xs">
      {managers.map((manager) => (
        <li key={`${manager.workspace}/${manager.manager}`} className="flex items-center gap-1.5">
          <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full", stateDot[manager.state])} />
          <span className="font-mono text-muted-foreground">{manager.workspace}</span>
          <span className={cn(manager.state === "ready" ? "text-muted-foreground" : "text-foreground")}>{managerLabel(manager)}</span>
        </li>
      ))}
    </ul>
  );
