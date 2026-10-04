import { Link } from "react-router";

import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { useFetchDocsByProject } from "@/hooks/DocHooks";
import { useInterviewTrails } from "@/hooks/InterviewSourceHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { usePlayRunStore } from "@/stores/playRunStore";
import { auditDocOf } from "@/models/Doc";
import { docPath, projectToken, type Project } from "@/models/Project";
import type { TrailState } from "@/models/Trail";

interface InterviewAuditLineProps {
  project: Project;
}

const STATE_LABEL: Record<TrailState, string> = {
  starting: "Auditing",
  running: "Auditing",
  waiting: "Auditing",
  done: "Audit written",
  failed: "Audit failed",
  interrupted: "Audit stopped",
};

// The latest audit run as one line: its trail icon, its state, and once it is done the doc it wrote.
export const InterviewAuditLine = ({ project }: InterviewAuditLineProps) => {
  const wsPath = useWorkspacePath();
  const latest = useInterviewTrails(project.id).audit?.[0];
  const state = usePlayRunStore((s) => (latest ? (s.frames[latest.id]?.state ?? latest.state) : undefined));
  const { data: docs } = useFetchDocsByProject(state === "done" ? project.id : "");
  const doc = latest && docs ? auditDocOf(latest, docs) : undefined;
  if (!latest || !state) return null;
  return (
    <p className="flex min-w-0 basis-full items-center gap-2 text-sm">
      <TrailStateIcon state={state} />
      <span className="shrink-0 font-medium">{STATE_LABEL[state]}</span>
      {doc && (
        <Link
          to={wsPath(docPath(projectToken(project), doc.id))}
          className="truncate text-muted-foreground underline-offset-4 transition-colors duration-150 ease-standard hover:text-foreground hover:underline"
        >
          · {doc.title}
        </Link>
      )}
    </p>
  );
};
