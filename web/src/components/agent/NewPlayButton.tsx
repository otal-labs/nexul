import { ActionButton } from "@/components/action/ActionButton";
import { usePlayDialogs } from "@/hooks/usePlayDialogs";
import type { AgentTab } from "@/models/Agent";

// New ticket play, New doc play and so on: the create form with the tab's type already set.
export const NewPlayButton = ({ tab }: { tab: AgentTab }) => {
  const { create } = usePlayDialogs();
  return (
    <ActionButton permission="plays:write" verb="create plays" type="button" onClick={() => void create(tab)}>
      New {tab} play
    </ActionButton>
  );
};
