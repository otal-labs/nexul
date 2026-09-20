import { WorkflowIcon } from "lucide-react";
import { Link } from "react-router";

import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";

interface CanvasEmptyHintProps {
  onAddNode: () => void;
}

// Wrapper ignores pointer events so pan/zoom still work underneath; only the CTAs are interactive.
export const CanvasEmptyHint = ({ onAddNode }: CanvasEmptyHintProps) => (
  <div className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center p-6">
    <EmptyState
      icon={WorkflowIcon}
      title="Your infra, drawn like you'd explain it."
      message="Sketch the pieces Nexul doesn't manage — domains, tunnels, databases — and deployed services will appear here on their own."
      action={
        <div className="pointer-events-auto flex flex-wrap items-center justify-center gap-3">
          <Button asChild>
            <Link to="/wizard/project/repository">Add a service</Link>
          </Button>
          <Button variant="outline" onClick={onAddNode}>
            Add a node
          </Button>
        </div>
      }
    />
  </div>
);
