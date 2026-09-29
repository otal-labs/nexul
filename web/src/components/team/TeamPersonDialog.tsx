import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { EmptyRow } from "@/components/EmptyRow";
import { PersonAvatar } from "@/components/PersonAvatar";
import { AccountStatusLabel } from "@/components/team/AccountStatusLabel";
import { TeamAccountActions } from "@/components/team/TeamAccountActions";
import { TeamAddToWorkspaceRow } from "@/components/team/TeamAddToWorkspaceRow";
import { TeamMembershipItem } from "@/components/team/TeamMembershipItem";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { personName, presenceText } from "@/utils/TeamUtility";

interface TeamPersonDialogProps {
  personId: string | null;
  onClose: () => void;
}

// Reads the person from the Team query the list already holds, so a change refreshes both at once.
export const TeamPersonDialog = ({ personId, onClose }: TeamPersonDialogProps) => {
  const { data: team } = useFetchTeam();
  const person = team?.people.find((candidate) => candidate.id === personId);
  const workspaceById = new Map(team?.workspaces.map((workspace) => [workspace.id, workspace]));

  return (
    <Dialog open={!!person} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="flex max-h-[min(90dvh,48rem)] flex-col gap-0 p-0 sm:max-w-[34rem]">
        {person && (
          <DialogHeader className="flex-row items-center gap-3 border-b border-border px-6 pt-6 pb-4 pr-12 text-left">
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
          <section aria-labelledby="team-person-workspaces" className="min-h-0 flex-1 space-y-3 overflow-y-auto px-6 py-4">
            <h3 id="team-person-workspaces" className="text-sm font-semibold">Workspaces</h3>
            {person.workspaces.length === 0 && <EmptyRow>Not a member of any workspace you can see.</EmptyRow>}
            {person.workspaces.length > 0 && (
              <ul className="divide-y divide-border rounded-md border border-border bg-card">
                {person.workspaces.map((membership) => {
                  const workspace = workspaceById.get(membership.workspace_id);
                  return workspace && <TeamMembershipItem key={membership.workspace_id} person={person} workspace={workspace} membership={membership} />;
                })}
              </ul>
            )}
            <TeamAddToWorkspaceRow person={person} workspaces={team.workspaces} />
          </section>
        )}
        {person && team && (
          <DialogFooter className="flex-row items-center justify-between border-t border-border px-6 py-3 sm:justify-between">
            {team.can_manage_accounts && <TeamAccountActions person={person} />}
            <DialogClose asChild>
              <Button type="button" size="sm" className="ml-auto">
                Done
              </Button>
            </DialogClose>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
};
