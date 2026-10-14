import { ArrowLeftRight, UserRound } from "lucide-react";

import { Logo } from "@/components/Logo";
import { microheaderClass } from "@/components/Microheader";
import { displayTitleClass } from "@/components/PageHeader";
import { PersonAvatar } from "@/components/PersonAvatar";
import type { RefusedPerson } from "@/models/Invitation";
import { cn } from "@/lib/utils";

interface InvitationHeaderProps {
  instanceName: string;
  host: string;
  person?: RefusedPerson | undefined;
}

// "This place and you": before sign-in the second mark is an empty seat, after it the person's avatar.
export const InvitationHeader = ({ instanceName, host, person }: InvitationHeaderProps) => (
  <header className="space-y-4 text-center">
    <div className="flex items-center justify-center gap-3">
      <Logo className="size-10 rounded-[10px] [&_svg]:size-6" />
      <ArrowLeftRight aria-hidden className="size-4 text-muted-foreground" />
      {person && <PersonAvatar login={person.login} src={person.avatar_url} className="size-10" />}
      {!person && (
        <span aria-hidden className="grid size-10 place-items-center rounded-full border border-dashed border-foreground/25 text-muted-foreground">
          <UserRound className="size-4" />
        </span>
      )}
    </div>
    <div className="space-y-1.5">
      <p className={microheaderClass}>Invitation</p>
      <h1 className={cn(displayTitleClass, "text-[1.75rem]")}>Join {instanceName}</h1>
      {host !== instanceName && <p className="font-mono text-xs break-all text-muted-foreground">{host}</p>}
    </div>
  </header>
);
