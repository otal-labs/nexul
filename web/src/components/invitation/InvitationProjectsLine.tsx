import { useState } from "react";
import { FolderKanban } from "lucide-react";

import type { InvitationProject } from "@/models/Invitation";

const SHOWN = 8;

// Names past the eighth fold behind "and N more", so a workspace with fifty projects stays one short line.
export const InvitationProjectsLine = ({ projects }: { projects: InvitationProject[] }) => {
  const [open, setOpen] = useState(false);
  const shown = open ? projects : projects.slice(0, SHOWN);
  const hidden = projects.length - shown.length;
  return (
    <p className="flex items-start gap-2 border-t bg-surface-2/60 px-3.5 py-2.5 text-xs leading-5 text-muted-foreground">
      <FolderKanban aria-hidden className="mt-0.5 size-3.5 shrink-0" />
      {projects.length === 0 && <span>No projects yet</span>}
      {projects.length > 0 && (
        <span className="min-w-0 break-words">
          <span className="text-foreground/85">{projects.length} {projects.length === 1 ? "project" : "projects"}:</span>{" "}
          {shown.map((project) => project.name).join(", ")}
          {hidden > 0 && " "}
          {hidden > 0 && (
            <button type="button" className="text-foreground/85 underline-offset-4 hover:underline" onClick={() => setOpen(true)}>
              and {hidden} more
            </button>
          )}
        </span>
      )}
    </p>
  );
};
