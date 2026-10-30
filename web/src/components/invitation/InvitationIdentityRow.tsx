import type { ReactNode } from "react";

import { PersonAvatar } from "@/components/PersonAvatar";
import type { RefusedPerson } from "@/models/Invitation";
import { providerLabel, loginHandle } from "@/models/User";
import { cn } from "@/lib/utils";

interface InvitationIdentityRowProps {
  person: RefusedPerson;
  // "Not you?", at the row's end.
  action?: ReactNode;
  className?: string;
}

// "Lena Fischer @lena · GitHub": who the invite page is talking to. The handle tells two people of one name apart, so a long
// name gives way and the handle drops to its own line rather than truncating.
export const InvitationIdentityRow = ({ person, action, className }: InvitationIdentityRowProps) => {
  // The server may name a provider this build has no label for, such as the development sign-in.
  const provider: string | undefined = providerLabel[person.provider];
  return (
    <div className={cn("flex min-w-0 flex-wrap items-start gap-x-2.5 gap-y-1", className)}>
      <PersonAvatar login={person.login} name={person.name} src={person.avatar_url} className="size-6 shrink-0" />
      <p className="flex min-w-0 flex-1 flex-wrap items-baseline gap-x-1.5 text-[13px] leading-6">
        <span className="max-w-full font-medium text-balance break-words">{person.name || person.login}</span>
        <span className="font-mono text-xs whitespace-nowrap text-muted-foreground">{loginHandle(person.login)}{provider && ` · ${provider}`}</span>
      </p>
      {action}
    </div>
  );
};
