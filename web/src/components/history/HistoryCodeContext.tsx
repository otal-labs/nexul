import { UnfoldVerticalIcon } from "lucide-react";
import { useState } from "react";

import { HistoryCodeLine } from "@/components/history/HistoryCodeLine";
import { foldContext, type CodeLine } from "@/utils/HistoryCodeUtility";

interface HistoryCodeContextProps {
  lines: CodeLine[];
  first: boolean;
  last: boolean;
  numbered: boolean;
}

export const HistoryCodeContext = ({ lines, first, last, numbered }: HistoryCodeContextProps) => {
  const [unfolded, setUnfolded] = useState(false);
  const fold = foldContext(lines, first, last, numbered);
  const shown = unfolded ? lines : [...fold.head, ...fold.tail];
  const at = fold.head.length;

  return (
    <>
      {shown.slice(0, at).map((line, i) => (
        <HistoryCodeLine key={i} line={line} numbered={numbered} />
      ))}
      {fold.hidden.length > 0 && !unfolded && (
        <button
          type="button"
          onClick={() => setUnfolded(true)}
          className="col-span-full flex items-center gap-2 bg-muted px-2 py-1 text-left text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <UnfoldVerticalIcon className="size-3.5" aria-hidden />
          {fold.hidden.length} unchanged lines
        </button>
      )}
      {shown.slice(at).map((line, i) => (
        <HistoryCodeLine key={at + i} line={line} numbered={numbered} />
      ))}
    </>
  );
};
