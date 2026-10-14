import { waitingCount } from "@nexul/client-core/clarification";
import { useState } from "react";

import { useFetchDocClarification } from "@/hooks/DocHooks";

export type DocView = "doc" | "questions";

// The side shown: the one the screen opened on, then the viewer's pick; with no round, or no questions route, only the doc.
export const useDocView = (docId: string, openedOn: DocView) => {
  const { data: clarification, isPending } = useFetchDocClarification(docId);
  const [chosen, setChosen] = useState<DocView>(openedOn);
  const switchable = !!clarification && clarification.rounds.length > 0;
  const waiting = clarification ? waitingCount(clarification) : 0;
  const view: DocView = chosen === "questions" && (switchable || isPending) ? "questions" : "doc";
  return { switchable, waiting, view, setView: setChosen };
};
