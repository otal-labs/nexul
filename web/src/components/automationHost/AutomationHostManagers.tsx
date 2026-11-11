import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
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

// One line per workspace token the host holds: the workspace's current name, then the manager's name and, when its token
// does not work, why. A workspace the viewer is not in is not named.
export const AutomationHostManagers = ({ managers }: AutomationHostManagersProps) => {
  const { data: workspaces } = useFetchWorkspaces(managers.length > 0);
  return (
    managers.length > 0 && (
      <ul aria-label="Secrets managers" className="mt-1 flex flex-wrap gap-x-4 gap-y-0.5 text-xs">
        {managers.map((manager) => (
          <li key={`${manager.workspace_id}/${manager.manager}`} className="flex items-center gap-1.5">
            <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full", stateDot[manager.state])} />
            <span className="text-muted-foreground">
              {workspaces?.find((w) => w.id === manager.workspace_id)?.name ?? (workspaces ? "Another workspace" : "")}
            </span>
            <span className={cn(manager.state === "ready" ? "text-muted-foreground" : "text-foreground")}>{managerLabel(manager)}</span>
          </li>
        ))}
      </ul>
    )
  );
};
