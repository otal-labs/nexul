import { menuItemClass } from "@/components/MenuItem";
import { PersonAvatar } from "@/components/PersonAvatar";
import { cn } from "@/lib/utils";
import type { TeamPerson } from "@/models/Team";
import { loginHandle } from "@/models/User";
import { personName } from "@/utils/TeamUtility";

interface RolePersonOptionProps {
  person: TeamPerson;
  disabled: boolean;
  onPick: (id: string) => void;
}

export const RolePersonOption = ({ person, disabled, onPick }: RolePersonOptionProps) => (
  <button type="button" className={cn(menuItemClass, "group/option")} disabled={disabled} onClick={() => onPick(person.id)}>
    <PersonAvatar login={person.login} name={personName(person)} src={person.avatar_url} className="size-6" />
    <span className="min-w-0 flex-1 truncate">
      {personName(person)}
      <span className="ml-2 font-mono text-xs text-muted-foreground">{loginHandle(person.login)}</span>
    </span>
    {/* A pick saves at once, so the row says what it does where the pointer or focus is. */}
    <span aria-hidden className="shrink-0 text-xs text-muted-foreground opacity-0 transition-opacity duration-150 ease-standard group-hover/option:opacity-100 group-focus-visible/option:opacity-100">
      Add
    </span>
  </button>
);
