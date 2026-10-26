import { ArrowLeftRight } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { InvitationIdentityRow } from "@/components/invitation/InvitationIdentityRow";
import { InvitationPanel } from "@/components/invitation/InvitationPanel";
import { Logo } from "@/components/Logo";
import { microheaderClass } from "@/components/Microheader";
import { displayTitleClass } from "@/components/PageHeader";
import { PersonAvatar } from "@/components/PersonAvatar";
import type { RefusedPerson } from "@/models/Invitation";
import { cn } from "@/lib/utils";

// The server never names the instance on a refusal, and the page is served from it.
const instanceHost = () => window.location.host;

// Says only that the link is gone: no inviter, no next step, nothing that tells expired from deleted.
export const InvitationGoneScreen = () => (
  <InvitationPanel className="text-center">
    <Logo className="mx-auto size-10 rounded-lg opacity-70 [&_svg]:size-6" />
    <p className={cn(microheaderClass, "mt-4")}>Invitation</p>
    <h1 className={cn(displayTitleClass, "mt-1.5 text-[1.5rem]")}>This invitation has been deleted or has expired</h1>
    <p className="mt-2 font-mono text-xs break-all text-muted-foreground">{instanceHost()}</p>
  </InvitationPanel>
);

// Turned away before anything is written, so the link stays unused for the person it was made for.
export const InvitationExistingAccountScreen = ({ person }: { person?: RefusedPerson | undefined }) => (
  <InvitationPanel className="text-center">
    <div className="flex items-center justify-center gap-3">
      <Logo className="size-10 rounded-lg [&_svg]:size-6" />
      <ArrowLeftRight aria-hidden className="size-4 text-muted-foreground" />
      {person && <PersonAvatar login={person.login} src={person.avatar_url} className="size-10" />}
    </div>
    <p className={cn(microheaderClass, "mt-4")}>Invitation</p>
    <h1 className={cn(displayTitleClass, "mt-1.5 text-[1.5rem]")}>You already have an account here</h1>
    <p className="mt-2 text-sm text-muted-foreground">This invitation is for someone new.</p>
    {person && <InvitationIdentityRow person={person} className="mt-6 justify-center" />}
    <Button asChild className="mt-4 w-full">
      <Link to="/" replace>Go to {instanceHost()}</Link>
    </Button>
    <p className="mt-4 text-xs text-muted-foreground">The link stays unused, so it still works for the person it was meant for.</p>
  </InvitationPanel>
);
