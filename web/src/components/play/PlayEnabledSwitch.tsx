import { ActionSwitch } from "@/components/action/ActionSwitch";
import { useUpdatePlay } from "@/hooks/PlayHooks";
import { useEntityActions } from "@/hooks/useEntityActions";
import { playToFormData, type Play } from "@/models/Play";

interface PlayEnabledSwitchProps {
  play: Play;
}

// Saves at once with the play's other values as they are, so a play's on or off never waits for the Edit form.
export const PlayEnabledSwitch = ({ play }: PlayEnabledSwitchProps) => {
  const answer = useEntityActions("play", play.id);
  const update = useUpdatePlay(play.workspace_id);

  return (
    <div className="flex items-center gap-2">
      <span aria-hidden className="text-sm text-muted-foreground">
        Enabled
      </span>
      <ActionSwitch
        permission="plays:write"
        answer={answer("plays:write")}
        verb="edit plays"
        aria-label="Enabled"
        checked={play.enabled}
        disabled={update.isPending}
        onCheckedChange={(enabled) => update.mutate({ playId: play.id, input: { ...playToFormData(play), enabled } })}
      />
    </div>
  );
};
