import { memo, useState } from "react";

import { AutoPlaySentence } from "@/components/autoplay/AutoPlaySentence";
import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { Switch } from "@/components/ui/switch";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useCreateAutoPlay, useDeleteAutoPlay, useUpdateAutoPlay } from "@/hooks/PlayHooks";
import { leavingRowClass } from "@/hooks/useRowGlide";
import { toRequest, type AutoPlay, type AutoPlaySubject } from "@/models/AutoPlay";
import { cn } from "@/lib/utils";

interface AutoPlayRowProps {
  autoPlay: AutoPlay;
  subject: AutoPlaySubject;
  canWrite: boolean;
  canDelete: boolean;
  onOpen: (autoPlay: AutoPlay) => void;
  // Called as the row starts to leave, so the list can glide the rows under it up once it is gone.
  onLeave: () => void;
}

export const AutoPlayRow = memo(({ autoPlay, subject, canWrite, canDelete, onOpen, onLeave }: AutoPlayRowProps) => {
  const update = useUpdateAutoPlay(autoPlay.workspace_id, autoPlay.play_id);
  const create = useCreateAutoPlay(autoPlay.workspace_id, autoPlay.play_id);
  const remove = useDeleteAutoPlay(autoPlay.workspace_id, autoPlay.play_id);
  const { open: confirm } = useConfirmationDialog();
  const [leaving, setLeaving] = useState(false);
  const name = `auto play ${autoPlay.id}`;

  const toggle = (enabled: boolean) => update.mutate({ id: autoPlay.id, input: { ...toRequest(autoPlay), enabled } });

  const destroy = async () => {
    const ok = await confirm({
      title: "Delete this auto play?",
      message: "The play stops starting itself at this moment. Runs it already started keep their trails.",
      confirmLabel: "Delete auto play",
    });
    if (!ok) return;
    setLeaving(true);
    onLeave();
    remove.mutate(autoPlay.id, { onError: () => setLeaving(false) });
  };

  const actions: RowAction[] = [
    { label: canWrite ? "Edit" : "View", onSelect: () => onOpen(autoPlay) },
    ...(canWrite ? [{ label: "Duplicate", onSelect: () => create.mutate({ ...toRequest(autoPlay), enabled: false }) }] : []),
    ...(canDelete ? [{ label: "Delete", destructive: true, onSelect: () => void destroy() }] : []),
  ];

  return (
    <li data-leaving={leaving || undefined} className={cn(leavingRowClass, "flex items-start gap-3 bg-card px-4 py-3 hover:bg-accent/40")}>
      <button
        type="button"
        onClick={() => onOpen(autoPlay)}
        className={cn("min-w-0 flex-1 text-left", !autoPlay.enabled && "opacity-60")}
      >
        <AutoPlaySentence autoPlay={autoPlay} subject={subject} />
      </button>
      <Switch
        checked={autoPlay.enabled}
        onCheckedChange={toggle}
        disabled={!canWrite}
        aria-label={`Switch ${name} on or off`}
        className="mt-0.5"
      />
      <div className="-my-1">
        <RowActionsMenu subject={name} actions={actions} />
      </div>
    </li>
  );
});
AutoPlayRow.displayName = "AutoPlayRow";
