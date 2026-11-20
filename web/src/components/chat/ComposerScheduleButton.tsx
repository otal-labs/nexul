import { CalendarClock } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useScheduleFormDialog } from "@/hooks/useScheduleFormDialog";

interface ComposerScheduleButtonProps {
  conversationId: string;
  // body is what the field holds, with its attachments, so the dialog opens on it.
  body: string;
  onScheduled: () => void;
}

// Opens the schedule dialog on what is typed; once it is scheduled the field empties, as a send does.
export const ComposerScheduleButton = ({ conversationId, body, onScheduled }: ComposerScheduleButtonProps) => {
  const { create } = useScheduleFormDialog(conversationId);
  const schedule = async () => {
    if (await create(body)) onScheduled();
  };
  return (
    <Button
      type="button"
      size="icon"
      variant="ghost"
      aria-label="Schedule message"
      title="Schedule message"
      className="text-muted-foreground hover:text-foreground"
      onClick={() => void schedule()}
    >
      <CalendarClock className="size-4" aria-hidden />
    </Button>
  );
};
