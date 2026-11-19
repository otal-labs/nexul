import { useState } from "react";
import { useSearchParams } from "react-router";

import { waitingCount } from "@nexul/client-core/clarification";

import { useFetchDocClarification } from "@/hooks/DocHooks";
import { useDocBuiltinPlays } from "@/hooks/PlayHooks";
import { useFetchTicketsByDoc } from "@/hooks/TicketHooks";
import type { Doc } from "@/models/Doc";

export type DocView = "doc" | "tickets" | "questions";

const VIEW_PARAM = "view";

// The doc page's view: the URL's ?view= when that view is offered, else Questions while some wait as the page opens.
export const useDocView = (doc: Doc) => {
  const [params, setParams] = useSearchParams();
  const { data: clarification } = useFetchDocClarification(doc.id);
  const { clarify } = useDocBuiltinPlays(doc.project_id);
  const { data: tickets } = useFetchTicketsByDoc(doc.id);
  // Picked once, as the clarification first arrives, so answering the last question never swaps the view away.
  const [opened, setOpened] = useState<DocView | null>(null);
  const waiting = clarification ? waitingCount(clarification) : 0;
  if (opened === null && clarification) setOpened(waiting > 0 ? "questions" : "doc");
  const ticketCount = tickets?.length ?? 0;
  const questions = !!clarification && (clarification.rounds.length > 0 || (clarification.can_close && !!clarify));
  const offered = (view: DocView) => view === "doc" || (view === "tickets" && ticketCount > 0) || (view === "questions" && questions);
  const asked = params.get(VIEW_PARAM) as DocView | null;
  const wanted = asked ?? opened ?? "doc";
  const view: DocView = offered(wanted) ? wanted : "doc";
  const setView = (next: DocView) =>
    setParams(
      (prev) => {
        const nextParams = new URLSearchParams(prev);
        nextParams.set(VIEW_PARAM, next);
        return nextParams;
      },
      { replace: true },
    );
  return { switchable: ticketCount > 0 || questions, questions, ticketCount, waiting, view, setView };
};
