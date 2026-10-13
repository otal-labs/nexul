import { FixLink } from "@/components/FixLink";
import { useRunBlock } from "@/hooks/PairingProjectHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { cn } from "@/lib/utils";

interface HarnessReadinessNoteProps {
  projectId: string;
  className?: string;
}

// Why no play can run here, said once for every play button in the section; each button only carries it as its tooltip.
export const HarnessReadinessNote = ({ projectId, className }: HarnessReadinessNoteProps) => {
  const canRun = useHasPermission("plays:run");
  const blocked = useRunBlock(projectId);
  if (!canRun || !blocked?.message) return null;
  return (
    <span className={cn("block text-xs text-muted-foreground", className)}>
      {blocked.message} {blocked.link && <FixLink {...blocked.link} />}
    </span>
  );
};
