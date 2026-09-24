import { useState } from "react";

import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { hasInterview } from "@/models/Memory";

// The project wizard's interview offer: pending until the project has an interview or the person confirms skipping it.
export const useInterviewOffer = (projectId: string | null, projectName: string) => {
  const { data: memories } = useFetchMemoriesByProject(projectId ?? "");
  const { open: confirm } = useConfirmationDialog();
  const [skipped, setSkipped] = useState(false);
  const pending = projectId !== null && !!memories && !hasInterview(memories, projectId) && !skipped;

  // Resolves true when the offer no longer holds anyone back: nothing pending, or the skip confirmed.
  const confirmSkip = async (): Promise<boolean> => {
    if (!pending) return true;
    const ok = await confirm({
      title: "Skip the interview?",
      message:
        `Without an interview, agents in ${projectName} work without its rules: the stack, paradigm, testing ` +
        "strategy, principles, and vocabulary the interview records. A cheaper model guesses where it would have " +
        "followed them. A banner stays on the project until the interview exists.",
      confirmLabel: "Skip the interview",
      cancelLabel: "Back",
      destructive: false,
    });
    if (ok) setSkipped(true);
    return ok;
  };

  return { pending, skipped, confirmSkip };
};
