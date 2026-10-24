import { useSearchParams } from "react-router";

import { MEETING_PARAM, SLOT_PARAM } from "@/models/Meeting";

// The open meeting lives in the address (?meeting=<id>&slot=<occurrence>), so a notification can link to its panel.
export const useMeetingParams = () => {
  const [params, setParams] = useSearchParams();

  const show = (meetingId: string, slot?: string) =>
    setParams((current) => {
      const next = new URLSearchParams(current);
      next.set(MEETING_PARAM, meetingId);
      if (slot) next.set(SLOT_PARAM, slot);
      else next.delete(SLOT_PARAM);
      return next;
    });

  const close = () =>
    setParams((current) => {
      const next = new URLSearchParams(current);
      next.delete(MEETING_PARAM);
      next.delete(SLOT_PARAM);
      return next;
    });

  return { meetingId: params.get(MEETING_PARAM) ?? "", slot: params.get(SLOT_PARAM) ?? "", show, close };
};
