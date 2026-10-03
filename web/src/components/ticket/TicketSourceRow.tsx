import { FileText, X } from "lucide-react";
import { Link } from "react-router";

import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchDoc } from "@/hooks/DocHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { docPath, projectTokenById } from "@/models/Project";

interface TicketSourceRowProps {
  docId: string;
  onRemove: () => void;
}

// The doc a ticket was derived from: its title links to the doc, and the × clears the source.
export const TicketSourceRow = ({ docId, onRemove }: TicketSourceRowProps) => {
  const { data: doc, error, isPending } = useFetchDoc(docId);
  const { data: projects } = useFetchProjects();
  const wsPath = useWorkspacePath();

  return (
    <li className="group flex items-center gap-2 rounded-md px-2 py-0.5 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <FileText className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
      {doc && projects && (
        <Link
          to={wsPath(docPath(projectTokenById(projects, doc.project_id), doc.id))}
          title={doc.title}
          className="mr-auto min-w-0 truncate py-1 text-xs underline-offset-4 hover:underline"
        >
          {doc.title}
        </Link>
      )}
      {error && <span className="mr-auto py-1 text-xs text-muted-foreground">Doc unavailable</span>}
      {isPending && <LoadingDisplay label="Loading doc…" className="mr-auto justify-start p-1 [&_span]:text-xs" />}
      <button
        type="button"
        aria-label="Remove source doc"
        onClick={onRemove}
        className="flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground opacity-0 transition-[color,background-color,opacity] duration-150 ease-standard group-focus-within:opacity-100 group-hover:opacity-100 hover:bg-muted/50 hover:text-foreground"
      >
        <X className="size-3.5" aria-hidden />
      </button>
    </li>
  );
};
