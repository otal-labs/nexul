import { ChannelsCard } from "@/components/accessgrid/ChannelsCard";
import { SettingsSaveBar } from "@/components/settings/SettingsSaveBar";
import { useSaveAccessGrid } from "@/hooks/AccessGridHooks";
import { useAccessDraft } from "@/hooks/useAccessDraft";
import { useFlash } from "@/hooks/useFlash";
import { countLabel, type AccessGrid } from "@/models/AccessGrid";

interface AccessChannelsTabProps {
  grid: AccessGrid;
}

const FORM = "access-channels";

// The Public channels grid saved from its own footer.
export const AccessChannelsTab = ({ grid }: AccessChannelsTabProps) => {
  const draft = useAccessDraft(grid);
  const save = useSaveAccessGrid(grid.workspace_id);
  const [saved, flash] = useFlash();
  const submit = async () => {
    await save.mutateAsync({ channels: draft.channelChanges });
    draft.clearChannels();
    flash();
  };
  return (
    <form
      id={FORM}
      className="flex min-h-0 flex-col"
      onSubmit={(event) => {
        event.preventDefault();
        void submit().catch(() => undefined);
      }}
    >
      <ChannelsCard
        grid={grid}
        fill
        footer={
          <SettingsSaveBar
            form={FORM}
            dirty={draft.channelChanges.length > 0}
            saving={save.isPending}
            saved={saved}
            onDiscard={draft.clearChannels}
            hint="Changes apply to everyone once saved."
            dirtyLabel={countLabel(draft.channelCells, "change", "changes")}
          />
        }
      />
    </form>
  );
};
