import { ClipboardListIcon } from "lucide-react";

import { EmptyState } from "@/components/EmptyState";

export const InterviewEmptyState = () => (
  <EmptyState
    icon={ClipboardListIcon}
    title="No interview yet"
    message="The interview holds this project's rules for agents and goes with every agent turn here. Run it to write them."
  />
);
