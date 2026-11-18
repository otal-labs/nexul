import { ArrowRight } from "lucide-react";

import type { ImportStatusMatch } from "@/models/Import";

interface ImportStatusRowProps {
  status: ImportStatusMatch;
}

const kindName: Record<string, string> = {
  backlog: "Open",
  progress: "In progress",
  review: "In review",
  testing: "Testing",
  done: "Closed",
};

// One source status and the column its tickets land in, saying when that column is new.
export const ImportStatusRow = ({ status }: ImportStatusRowProps) => (
  <li className="flex min-h-11 items-center gap-3 py-2 text-sm">
    <span className="min-w-0 flex-1 truncate">{status.name || kindName[status.kind] || status.kind}</span>
    <ArrowRight className="size-3.5 shrink-0 text-muted-foreground" aria-label="lands in" />
    <span className="min-w-0 flex-1 truncate">
      {status.column}
      {status.new && <span className="text-muted-foreground"> · new column</span>}
    </span>
  </li>
);
