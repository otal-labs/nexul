import { PersonAvatar } from "@/components/PersonAvatar";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { AccountStatusLabel } from "@/components/team/AccountStatusLabel";
import { TeamAccountActions } from "@/components/team/TeamAccountActions";
import { TeamWorkspaceAccessRow } from "@/components/team/TeamWorkspaceAccessRow";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { personName } from "@/utils/TeamUtility";

interface TeamPersonSheetProps {
  personId: string | null;
  onClose: () => void;
}

// Reads the person from the Team query the list already holds, so a change refreshes both at once.
export const TeamPersonSheet = ({ personId, onClose }: TeamPersonSheetProps) => {
  const { data: team } = useFetchTeam();
  const person = team?.people.find((candidate) => candidate.id === personId);

  return (
    <Sheet open={!!person} onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="w-full gap-0 overflow-y-auto sm:max-w-xl">
        {person && (
          <SheetHeader className="border-b border-border">
            <div className="flex items-center gap-3 pr-8">
              <PersonAvatar login={person.login} src={person.avatar_url} className="size-10" />
              <div className="min-w-0 flex-1">
                <SheetTitle className="truncate">{personName(person)}</SheetTitle>
                <SheetDescription className="truncate font-mono text-xs">@{person.login}</SheetDescription>
              </div>
              <AccountStatusLabel status={person.status} />
            </div>
            {team?.can_manage_accounts && <TeamAccountActions person={person} />}
          </SheetHeader>
        )}
        {person && team && (
          <section aria-labelledby="team-workspace-access" className="space-y-3 p-4">
            <div>
              <h3 id="team-workspace-access" className="text-sm font-semibold">Workspace access</h3>
              <p className="text-sm text-muted-foreground">You can change the workspaces where you manage members.</p>
            </div>
            <ul className="divide-y divide-border overflow-hidden rounded-md border bg-card">
              {team.workspaces.map((workspace) => (
                <TeamWorkspaceAccessRow key={workspace.id} person={person} workspace={workspace} />
              ))}
            </ul>
          </section>
        )}
      </SheetContent>
    </Sheet>
  );
};
