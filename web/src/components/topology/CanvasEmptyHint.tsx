import { WorkflowIcon } from "lucide-react";
import { Link } from "react-router";

import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

interface CanvasEmptyHintProps {
  onAddNode: () => void;
}

// Wrapper ignores pointer events so pan/zoom still work underneath; only the CTAs are interactive.
export const CanvasEmptyHint = ({ onAddNode }: CanvasEmptyHintProps) => {
  const canAddService = useAreaAccess()?.("newProject") ?? false;
  const wsPath = useWorkspacePath();
  return (
    <div className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center p-6">
      <EmptyState
        icon={WorkflowIcon}
        title="Nothing on the canvas yet"
        message="Deployed services show up here by themselves. Add nodes for what Nexul doesn't run, like domains or databases."
        action={
          <div className="pointer-events-auto flex flex-wrap items-center justify-center gap-3">
            {canAddService && (
              <Button asChild>
                <Link to={wsPath("/wizard/project/repository")}>Add a service</Link>
              </Button>
            )}
            <Button variant="outline" onClick={onAddNode}>
              Add a node
            </Button>
          </div>
        }
      />
    </div>
  );
};
