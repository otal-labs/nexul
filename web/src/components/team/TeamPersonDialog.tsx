import { useReducer, useState } from "react";
import { toast } from "sonner";

import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { PersonAvatar } from "@/components/PersonAvatar";
import { AccountStatusLabel } from "@/components/team/AccountStatusLabel";
import { TeamDialogFooter } from "@/components/team/TeamDialogFooter";
import { TeamWorkspaceTabs } from "@/components/team/TeamWorkspaceTabs";
import { useApplyMemberStep, useFetchTeam } from "@/hooks/TeamHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { MemberDraftContext } from "@/hooks/useMemberDraft";
import { errorMessage } from "@/api/client";
import { hasOverlap, memberDraftReducer, memberSteps, type MemberStep } from "@/models/MemberDraft";
import { personName, presenceText } from "@/utils/TeamUtility";

interface TeamPersonDialogProps {
  personId: string | null;
  onClose: () => void;
}

const stepText = (step: MemberStep): string => {
  if (step.kind === "add") return `add to ${step.workspaceName}`;
  if (step.kind === "remove") return `remove from ${step.workspaceName}`;
  return `update ${step.workspaceName}`;
};

// Every workspace change is held until Confirm; account status in the footer still applies on its own.
export const TeamPersonDialog = ({ personId, onClose }: TeamPersonDialogProps) => {
  const { data: team } = useFetchTeam();
  const person = team?.people.find((candidate) => candidate.id === personId);
  const [draft, dispatch] = useReducer(memberDraftReducer, {});
  const [failure, setFailure] = useState<string>();
  const [applying, setApplying] = useState(false);
  const apply = useApplyMemberStep();
  const { open: ask } = useConfirmationDialog();
  const steps = person && team ? memberSteps(person, team.workspaces, draft) : [];
  const blocked = !!person && hasOverlap(person, draft);

  const finish = () => {
    dispatch({ type: "reset" });
    setFailure(undefined);
    onClose();
  };

  const close = async () => {
    if (steps.length > 0) {
      const discard = await ask({ title: "Discard changes?", message: "What you changed here hasn't been confirmed and will be lost.", confirmLabel: "Discard" });
      if (!discard) return;
    }
    finish();
  };

  // Stops at the first refusal and keeps whatever has not applied yet, so a second Confirm resends only that.
  const confirm = async () => {
    if (!person) return;
    setApplying(true);
    setFailure(undefined);
    for (const step of steps) {
      try {
        await apply.mutateAsync({ userId: person.id, step });
      } catch (error) {
        setFailure(`Couldn't ${stepText(step)}: ${errorMessage(error)}`);
        setApplying(false);
        return;
      }
      dispatch({ type: "applied", step });
    }
    setApplying(false);
    toast.success(`${personName(person)}'s access saved`);
    finish();
  };

  return (
    <Dialog open={!!person} onOpenChange={(open) => !open && void close()}>
      <DialogContent className="flex max-h-[min(90dvh,52rem)] flex-col gap-0 p-0 sm:max-w-[40rem]">
        {person && (
          <DialogHeader className="flex-row items-center gap-3 px-6 pt-6 pb-4 pr-12 text-left">
            <PersonAvatar login={person.login} src={person.avatar_url} className="size-10" />
            <div className="min-w-0 flex-1 space-y-1">
              <DialogTitle className="truncate">{personName(person)}</DialogTitle>
              <DialogDescription className="truncate font-mono text-xs">@{person.login}</DialogDescription>
              <p className="flex items-center gap-3 text-sm text-muted-foreground">
                {presenceText(person)}
                <AccountStatusLabel status={person.status} online={person.online} />
              </p>
            </div>
          </DialogHeader>
        )}
        {person && team && (
          <MemberDraftContext value={{ person, draft, dispatch }}>
            <TeamWorkspaceTabs workspaces={team.workspaces} steps={steps} />
          </MemberDraftContext>
        )}
        {person && team && (
          <TeamDialogFooter
            person={person}
            failure={failure}
            canConfirm={steps.length > 0 && !blocked}
            applying={applying}
            onCancel={() => void close()}
            onConfirm={() => void confirm()}
          />
        )}
      </DialogContent>
    </Dialog>
  );
};
