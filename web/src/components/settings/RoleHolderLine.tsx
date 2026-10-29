import { DialogDescription } from "@/components/ui/dialog";
import { PersonAvatar } from "@/components/PersonAvatar";
import type { TeamPerson } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

interface RoleHolderLineProps {
  holders: TeamPerson[];
}

const holderText = (holders: TeamPerson[]): string => {
  if (holders.length === 0) return "Nobody holds this role yet.";
  const names = holders.map(personName);
  if (names.length > 2) return `Held by ${names.slice(0, 2).join(", ")} and ${names.length - 2} more. Saving changes what they can do.`;
  if (names.length === 2) return `Held by ${names.join(" and ")}. Saving changes what they can do.`;
  return `Held by ${names[0]}. Saving changes what they can do.`;
};

// Who a save reaches, under the title of an edit.
export const RoleHolderLine = ({ holders }: RoleHolderLineProps) => (
  <DialogDescription className="flex items-start gap-2">
    {holders.length > 0 && (
      <span className="flex h-5 shrink-0 items-center -space-x-1.5">
        {holders.slice(0, 4).map((person) => (
          <PersonAvatar
            key={person.id}
            login={person.login}
            src={person.avatar_url}
            label={personName(person)}
            className="size-5 text-[10px] ring-2 ring-popover"
          />
        ))}
      </span>
    )}
    <span className="min-w-0 leading-5 text-pretty">{holderText(holders)}</span>
  </DialogDescription>
);
