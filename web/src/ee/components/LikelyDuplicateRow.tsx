import { ArrowRight } from "lucide-react";

import { Checkbox } from "@/components/ui/checkbox";
import { SignInMovePerson } from "@/ee/components/SignInMovePerson";
import type { MoveAccount, OldSignIn } from "@/ee/models/SignInMove";
import { formatShortDate } from "@/utils/TimeUtility";

interface LikelyDuplicateRowProps {
  person: OldSignIn & { match: MoveAccount };
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}

// One pair: the person still on the other protocol, and the account a first sign-in through the saved one made under their email.
export const LikelyDuplicateRow = ({ person, checked, onCheckedChange }: LikelyDuplicateRowProps) => (
  <li className="flex items-center gap-3 py-3">
    <Checkbox
      checked={checked}
      onCheckedChange={(value) => onCheckedChange(value === true)}
      aria-label={`Link ${person.match.email} to ${person.name || person.login}`}
    />
    <SignInMovePerson account={person} className="flex-1" />
    <ArrowRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
    <span className="flex min-w-0 flex-1 items-center justify-between gap-3">
      <SignInMovePerson account={person.match} />
      <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">made {formatShortDate(person.match.created_at)}</span>
    </span>
  </li>
);
