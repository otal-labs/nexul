import { useFetchTicketLinkSet } from "@/hooks/TicketLinkHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { linkedTicketKey } from "@/models/TicketLink";
import type { PlayType } from "@/models/Play";

// Blocked-by is a signal, never a gate: a play on a blocked ticket asks first, and a cancel is the only thing that stops it.
export const useConfirmBlockedRun = (targetType: PlayType, targetId: string) => {
  const { data: links } = useFetchTicketLinkSet(targetType === "ticket" ? targetId : undefined);
  const { open: confirm } = useConfirmationDialog();
  return async (playLabel: string): Promise<boolean> => {
    if (!links?.blocked) return true;
    const waiting = links.blocked_by.filter((t) => !t.done).map(linkedTicketKey);
    return confirm({
      title: "This ticket is blocked",
      message: `It still waits on ${waiting.join(", ")}. Run ${playLabel} anyway?`,
      confirmLabel: "Run anyway",
      destructive: false,
    });
  };
};
