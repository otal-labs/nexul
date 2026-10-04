import type { ReactNode } from "react";
import { LockIcon } from "lucide-react";
import { Link } from "react-router";

import { DocThreadButton } from "@/components/chat/DocThreadButton";
import { DocWatchButton } from "@/components/doc/DocWatchButton";
import { PlaysMenu } from "@/components/play/PlaysMenu";
import { useClarifyDev, useClarifyPrototypeStore } from "@/components/doc/prototype/ClarifyPrototypeStore";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Doc } from "@/models/Doc";

interface ClarifyPrototypeHeaderProps {
  doc: Doc;
  start?: ReactNode;
  end?: ReactNode;
}

// The doc page's header bar, with a slot after the back link and one before the actions.
export const ClarifyPrototypeHeader = ({ doc, start, end }: ClarifyPrototypeHeaderProps) => {
  const dev = useClarifyDev();
  const locked = useClarifyPrototypeStore((s) => s.phase === "running");
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const wsPath = useWorkspacePath();
  return (
    <div className="flex flex-wrap items-center justify-between gap-2">
      <div className="flex items-center gap-3">
        <Link
          to={wsPath("/docs")}
          className="font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground lg:hidden"
        >
          ← All docs
        </Link>
        {start}
      </div>
      <div className="flex items-center gap-1">
        {end}
        {locked && (
          <span className="flex items-center gap-1.5 px-1 font-mono text-xs text-muted-foreground">
            <LockIcon className="size-3.5" aria-hidden />
            Locked
          </span>
        )}
        {dev && <PlaysMenu workspaceId={workspaceId} projectId={doc.project_id} docId={doc.id} />}
        <DocWatchButton docId={doc.id} />
        {dev && <DocThreadButton workspaceId={workspaceId} docId={doc.id} />}
      </div>
    </div>
  );
};
