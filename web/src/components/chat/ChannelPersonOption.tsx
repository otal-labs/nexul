import { Checkbox } from "@/components/ui/checkbox";
import { PersonAvatar } from "@/components/PersonAvatar";
import { personLabel, type Person } from "@/models/Person";

interface ChannelPersonOptionProps {
  person: Person;
  isYou: boolean;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}

export const ChannelPersonOption = ({ person, isYou, checked, onCheckedChange }: ChannelPersonOptionProps) => (
  <li>
    <label className="flex min-h-11 cursor-pointer items-center gap-3 rounded-md px-2 transition-colors duration-150 ease-standard hover:bg-accent/40 has-[:disabled]:cursor-default has-[:disabled]:hover:bg-transparent">
      <Checkbox checked={checked} disabled={isYou} onCheckedChange={(on) => onCheckedChange(on === true)} aria-label={personLabel(person)} />
      <PersonAvatar login={person.login} src={person.avatar_url} className="size-6 text-xs" />
      <span className="min-w-0 flex-1 truncate text-sm">{personLabel(person)}</span>
      {isYou && <span className="font-mono text-xs text-muted-foreground">you</span>}
    </label>
  </li>
);
