import { canChangeMeeting, type Meeting } from "@nexul/client-core/calendar";

import { useFetchMe } from "@/hooks/AuthHooks";
import { useActionStates } from "@/hooks/useActionState";

// The organiser with calendar:write or anyone with calendar:delete: an either-or that one action gate can't say.
export const useCanChangeMeeting = (meeting: Meeting): boolean => {
  const { data: me } = useFetchMe();
  const [write, remove] = useActionStates([{ permission: "calendar:write" }, { permission: "calendar:delete" }]);
  const holds = (action: "calendar:write" | "calendar:delete") => (action === "calendar:write" ? write : remove)?.state === "live";
  return !!me && canChangeMeeting(meeting, me.user.id, holds);
};
