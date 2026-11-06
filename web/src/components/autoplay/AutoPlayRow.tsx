import { memo, useState } from "react";

import { AutoPlayClauses } from "@/components/autoplay/AutoPlayClauses";
import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { ActionSwitch } from "@/components/action/ActionSwitch";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useCreateAutoPlay, useDeleteAutoPlay, useUpdateAutoPlay } from "@/hooks/PlayHooks";
import { leavingRowClass } from "@/hooks/useRowGlide";
import { toRequest, type AutoPlay, type AutoPlaySubject } from "@/models/AutoPlay";
import { cn } from "@/lib/utils";

interface AutoPlayRowProps {
  autoPlay: AutoPlay;
  subject: AutoPlaySubject;
  canWrite: boolean;
  onOpen: (autoPlay: AutoPlay) => void;
  // Called as the row starts to leave, so the list can glide the rows under it up once it is gone.
  onLeave: () => void;
}

export const AutoPlayRow = memo(({ autoPlay, subject, canWrite, onOpen, onLeave }: AutoPlayRowProps) => {
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
    { label: "Duplicate", permission: "autoplays:write", verb: "duplicate auto plays", onSelect: () => create.mutate({ ...toRequest(autoPlay), enabled: false }) },
    { label: "Delete", destructive: true, permission: "autoplays:delete", onSelect: () => void destroy() },
  ];

  return (
    <li
      data-leaving={leaving || undefined}
      data-off={!autoPlay.enabled || undefined}
      className={cn(leavingRowClass, "group/row flex items-start gap-3 bg-card pr-4 hover:bg-accent")}
    >
      <button type="button" onClick={() => onOpen(autoPlay)} className="inset-focus min-w-0 flex-1 rounded-md py-3 pl-4 text-left">
        <AutoPlayClauses autoPlay={autoPlay} subject={subject} />
      </button>
      {/* One 20px box on the moment's line, so the switch and the menu centre on it however far the clauses wrap. */}
      <div className="mt-3 flex h-5 shrink-0 items-center gap-1">
        <ActionSwitch
          permission="autoplays:write"
          verb="switch auto plays on or off"
          checked={autoPlay.enabled}
          onCheckedChange={toggle}
          aria-label={`Switch ${name} on or off`}
        />
        <RowActionsMenu subject={name} actions={actions} />
      </div>
    </li>
  );
});
AutoPlayRow.displayName = "AutoPlayRow";
