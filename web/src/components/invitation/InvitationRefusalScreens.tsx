import type { ReactNode } from "react";
import { ArrowLeftRight } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { Logo } from "@/components/Logo";
import { microheaderClass } from "@/components/Microheader";
import { displayTitleClass } from "@/components/PageHeader";
import { PersonAvatar } from "@/components/PersonAvatar";
import type { RefusedPerson } from "@/models/Invitation";
import { providerLabel } from "@/models/User";
import { cn } from "@/lib/utils";

// The server never names the instance on a refusal, and the page is served from it.
const instanceHost = () => window.location.host;

const RefusalFrame = ({ children }: { children: ReactNode }) => (
  <div className="flex min-h-full items-center justify-center px-4 py-10 sm:px-6 sm:py-12">
    <div className="panel w-full max-w-[28rem] p-6 text-center [--panel-opacity:72%] sm:p-8">{children}</div>
  </div>
);

// Says only that the link is gone: no inviter, no next step, nothing that tells expired from deleted.
export const InvitationGoneScreen = () => (
  <RefusalFrame>
    <Logo className="mx-auto size-10 rounded-[10px] opacity-70 [&_svg]:size-6" />
    <p className={cn(microheaderClass, "mt-4")}>Invitation</p>
    <h1 className={cn(displayTitleClass, "mt-1.5 text-[1.5rem]")}>This invitation has been deleted or has expired</h1>
    <p className="mt-2 font-mono text-xs break-all text-muted-foreground">{instanceHost()}</p>
  </RefusalFrame>
);

const RefusedIdentityRow = ({ person }: { person: RefusedPerson }) => {
  // The server may name a provider this build has no label for, such as the development sign-in.
  const provider: string | undefined = providerLabel[person.provider];
  return (
    <div className="mt-6 flex min-w-0 items-center justify-center gap-2.5">
      <PersonAvatar login={person.login} src={person.avatar_url} className="size-6" />
      <p className="min-w-0 truncate text-[13px]">
        <span className="font-medium">{person.name || person.login}</span>
        <span className="font-mono text-xs text-muted-foreground"> @{person.login}{provider && ` · ${provider}`}</span>
      </p>
    </div>
  );
};

// Turned away before anything is written, so the link stays unused for the person it was made for.
export const InvitationExistingAccountScreen = ({ person }: { person?: RefusedPerson | undefined }) => (
  <RefusalFrame>
    <div className="flex items-center justify-center gap-3">
      <Logo className="size-10 rounded-[10px] [&_svg]:size-6" />
      <ArrowLeftRight aria-hidden className="size-4 text-muted-foreground" />
      {person && <PersonAvatar login={person.login} src={person.avatar_url} className="size-10" />}
    </div>
    <p className={cn(microheaderClass, "mt-4")}>Invitation</p>
    <h1 className={cn(displayTitleClass, "mt-1.5 text-[1.5rem]")}>You already have an account here</h1>
    <p className="mt-2 text-sm text-muted-foreground">This invitation is for someone new.</p>
    {person && <RefusedIdentityRow person={person} />}
    <Button asChild className="mt-4 w-full">
      <Link to="/" replace>Go to {instanceHost()}</Link>
    </Button>
    <p className="mt-4 text-xs text-muted-foreground">The link stays unused, so it still works for the person it was meant for.</p>
  </RefusalFrame>
);
