import { PersonAvatar } from "@/components/PersonAvatar";
import type { RefusedPerson } from "@/models/Invitation";
import { providerLabel, loginHandle } from "@/models/User";
import { cn } from "@/lib/utils";

// "Lena Fischer @lena · GitHub": who the invite page is talking to.
export const InvitationIdentityRow = ({ person, className }: { person: RefusedPerson; className?: string }) => {
  // The server may name a provider this build has no label for, such as the development sign-in.
  const provider: string | undefined = providerLabel[person.provider];
  return (
    <div className={cn("flex min-w-0 items-center gap-2.5", className)}>
      <PersonAvatar login={person.login} src={person.avatar_url} className="size-6" />
      <p className="min-w-0 truncate text-[13px]">
        <span className="font-medium">{person.name || person.login}</span>
        <span className="font-mono text-xs text-muted-foreground"> {loginHandle(person.login)}{provider && ` · ${provider}`}</span>
      </p>
    </div>
  );
};
