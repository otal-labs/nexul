import { ActionSwitch } from "@/components/action/ActionSwitch";
import { useSetPlayEnabled } from "@/hooks/PlayHooks";
import type { WorkspacePlay } from "@/models/Play";

// Saves at once with the play's other values as they are.
export const PlayEnabledSwitch = ({ play }: { play: WorkspacePlay }) => {
  const update = useSetPlayEnabled();
  return (
    <ActionSwitch
      permission="plays:write"
      verb="edit plays"
      label={`${play.label} on or off`}
      value={play.enabled}
      disabled={update.isPending}
      onValueChange={(enabled) => update.mutate({ play, enabled })}
    />
  );
};
