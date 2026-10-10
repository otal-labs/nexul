import { useHarnessReadiness } from "@/hooks/PairingHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { canChooseRunLocation } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface HarnessReadinessNoteProps {
  projectId: string;
  className?: string;
}

// Why no play can run here, said once for every play button in the section; each button only carries it as its tooltip.
export const HarnessReadinessNote = ({ projectId, className }: HarnessReadinessNoteProps) => {
  const canRun = useHasPermission("plays:run");
  const readiness = useHarnessReadiness(projectId);
  if (!canRun || !readiness || readiness.state === "ready" || canChooseRunLocation(readiness)) return null;
  return <span className={cn("block text-xs text-muted-foreground", className)}>{readiness.message}</span>;
};
