import type { ImportFailure } from "@/models/Import";

interface ImportFailureRowProps {
  failure: ImportFailure;
}

export const ImportFailureRow = ({ failure }: ImportFailureRowProps) => (
  <li className="py-2">
    <p className="truncate text-sm">{failure.title || failure.source_id}</p>
    <p className="text-xs text-destructive">{failure.error}</p>
  </li>
);
