import { useState } from "react";

import { useFetchDocClarification } from "@/hooks/DocHooks";
import { useDocBuiltinPlays } from "@/hooks/PlayHooks";
import { waitingCount } from "@/models/Clarification";
import type { Doc } from "@/models/Doc";

export type DocView = "doc" | "questions";

// Which of the doc page's two views shows: Questions when some wait as the page opens, then the viewer's pick; the
// switch hides when the doc has no clarification and the viewer can't start one.
export const useDocView = (doc: Doc) => {
  const { data: clarification } = useFetchDocClarification(doc.id);
  const { clarify } = useDocBuiltinPlays(doc.project_id);
  const [chosen, setChosen] = useState<DocView | null>(null);
  const waiting = clarification ? waitingCount(clarification) : 0;
  const switchable = !!clarification && (clarification.rounds.length > 0 || (clarification.can_close && !!clarify));
  // Picked once, as the clarification first arrives, so answering the last question never swaps the view away.
  if (chosen === null && clarification) setChosen(waiting > 0 ? "questions" : "doc");
  const view: DocView = switchable ? (chosen ?? "doc") : "doc";
  return { switchable, waiting, view, setView: setChosen };
};
