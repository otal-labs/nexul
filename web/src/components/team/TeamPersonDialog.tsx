import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { PersonAvatar } from "@/components/PersonAvatar";
import { AccountStatusLabel } from "@/components/team/AccountStatusLabel";
import { TeamDialogFooter } from "@/components/team/TeamDialogFooter";
import { TeamWelcomeBlock } from "@/components/team/TeamWelcomeBlock";
import { TeamWorkspaceTabs } from "@/components/team/TeamWorkspaceTabs";
import { TeamSsoRow } from "@/ee/components/TeamSsoRow";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { usePersonNamer } from "@/hooks/usePersonNamer";
import { personName, presenceText } from "@/utils/TeamUtility";
import { formatRelativeTime } from "@/utils/TimeUtility";
import { loginHandle } from "@/models/User";

interface TeamPersonDialogProps {
  personId: string | null;
  onClose: () => void;
}

// The account and where it is: its status actions apply at once, and its roles show per workspace, read-only.
export const TeamPersonDialog = ({ personId, onClose }: TeamPersonDialogProps) => {
  const { data: team } = useFetchTeam();
  const person = team?.people.find((candidate) => candidate.id === personId);
  const nameOf = usePersonNamer();

  return (
    <Dialog open={!!person} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="flex max-h-[min(90dvh,52rem)] flex-col gap-0 p-0 sm:max-w-[40rem]">
        {person && (
          <DialogHeader className="flex-row items-center gap-3 px-6 pt-6 pb-4 pr-12 text-left">
            <PersonAvatar login={person.login} name={personName(person)} src={person.avatar_url} className="size-10" />
            <div className="min-w-0 flex-1 space-y-1">
              <DialogTitle dir="auto" className="line-clamp-2 break-words">{personName(person)}</DialogTitle>
              <DialogDescription className="truncate font-mono text-xs">{loginHandle(person.login)}</DialogDescription>
              <p className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
                {presenceText(person)}
                {person.joined && (
                  <span>
                    Joined {formatRelativeTime(person.joined.accepted_at)}, invited by {nameOf(person.joined.invited_by)}
                  </span>
                )}
                <AccountStatusLabel status={person.status} />
              </p>
            </div>
          </DialogHeader>
        )}
        {person && <TeamSsoRow person={person} />}
        {person && <TeamWelcomeBlock person={person} />}
        {person && team && <TeamWorkspaceTabs person={person} workspaces={team.workspaces} />}
        {person && <TeamDialogFooter person={person} onClose={onClose} />}
      </DialogContent>
    </Dialog>
  );
};
