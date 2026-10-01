import { Button } from "@/components/ui/button";
import { useMemberDraft } from "@/hooks/useMemberDraft";
import type { TeamWorkspace } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

interface TeamRemovedNoticeProps {
  workspace: TeamWorkspace;
}

// A removal held for Confirm, with its way back before it applies.
export const TeamRemovedNotice = ({ workspace }: TeamRemovedNoticeProps) => {
  const { person, dispatch } = useMemberDraft();
  return (
    <div className="flex flex-wrap items-center gap-3 rounded-md border border-dashed border-border px-3 py-3">
      <p className="min-w-0 flex-1 basis-56 text-sm text-muted-foreground">
        {personName(person)} leaves {workspace.name} when you confirm. Their account and what they wrote stay, and you can add them back later.
      </p>
      <Button type="button" variant="outline" size="sm" onClick={() => dispatch({ type: "remove", workspaceId: workspace.id, removed: false })}>
        Keep in workspace
      </Button>
    </div>
  );
};
