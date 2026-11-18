import { createContext, use } from "react";

// The slot above a thread's composer: an open question renders into it, so it waits where the reply is typed, not up the transcript.
export const QuestionDockContext = createContext<HTMLElement | null>(null);

export const useQuestionDock = (): HTMLElement | null => use(QuestionDockContext);
