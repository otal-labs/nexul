import { menuItemClass } from "@/components/MenuItem";
import { PersonAvatar } from "@/components/PersonAvatar";
import type { TeamPerson } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

interface RolePersonOptionProps {
  person: TeamPerson;
  disabled: boolean;
  onPick: (id: string) => void;
}

export const RolePersonOption = ({ person, disabled, onPick }: RolePersonOptionProps) => (
  <button type="button" className={menuItemClass} disabled={disabled} onClick={() => onPick(person.id)}>
    <PersonAvatar login={person.login} name={personName(person)} src={person.avatar_url} className="size-5" />
    <span className="min-w-0 flex-1 truncate">{personName(person)}</span>
  </button>
);
