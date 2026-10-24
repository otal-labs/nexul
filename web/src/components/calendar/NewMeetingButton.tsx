import { PlusIcon } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { useMeetingFormDialog } from "@/hooks/useMeetingFormDialog";

// Everyone sees it; without calendar:write it is locked with the reason, where the role shows locked actions.
export const NewMeetingButton = () => {
  const { create } = useMeetingFormDialog();
  return (
    <ActionButton permission="calendar:write" verb="create meetings" icon={PlusIcon} onClick={() => void create()}>
      New meeting
    </ActionButton>
  );
};
