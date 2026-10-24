import { Navigate } from "react-router";

import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { meetingPath } from "@/models/Meeting";

// A meeting notice has nothing to show beside the list: it opens the calendar with the meeting's panel.
export const MeetingNoticeDetail = ({ meetingId }: { meetingId: string }) => {
  const wsPath = useWorkspacePath();
  return <Navigate to={wsPath(meetingPath(meetingId))} replace />;
};
