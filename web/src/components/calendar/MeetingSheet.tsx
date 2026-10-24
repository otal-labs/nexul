import { useState } from "react";

import { MeetingPanel } from "@/components/calendar/MeetingPanel";
import { Sheet, SheetContent } from "@/components/ui/sheet";
import { useMeetingParams } from "@/hooks/useMeetingParams";

// The meeting named in the address opens beside the calendar; closing it clears the address.
export const MeetingSheet = () => {
  const { meetingId, slot, close } = useMeetingParams();
  // The panel keeps showing its meeting while it slides out, after the address has let go of it.
  const [shown, setShown] = useState({ meetingId, slot });
  if (meetingId !== "" && (meetingId !== shown.meetingId || slot !== shown.slot)) setShown({ meetingId, slot });
  return (
    <Sheet open={meetingId !== ""} onOpenChange={(open) => !open && close()}>
      <SheetContent className="w-full gap-0 sm:max-w-md">
        <MeetingPanel meetingId={shown.meetingId} slot={shown.slot} />
      </SheetContent>
    </Sheet>
  );
};
