import { Logo } from "@/components/Logo";
import { microheaderClass } from "@/components/Microheader";
import { displayTitleClass } from "@/components/PageHeader";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { ProtoInvite } from "@/proto/invitations/data";
import { MarkPair, SignedInAs } from "@/proto/invitations/parts";

const Frame = ({ children }: { children: React.ReactNode }) => (
  <div className="flex min-h-full items-center justify-center px-4 py-10 sm:px-6 sm:py-12">
    <div className="panel w-full max-w-[28rem] p-6 text-center [--panel-opacity:72%] sm:p-8">{children}</div>
  </div>
);

// Says only that the link is gone: no inviter, no next step, nothing that confirms who made it.
export const ExpiredScreen = ({ invite }: { invite: ProtoInvite }) => (
  <Frame>
    <Logo className="mx-auto size-10 rounded-[10px] opacity-70 [&_svg]:size-6" />
    <p className={cn(microheaderClass, "mt-4")}>Invitation</p>
    <h1 className={cn(displayTitleClass, "mt-1.5 text-[1.5rem]")}>This invitation has been deleted or has expired</h1>
    <p className="mt-2 font-mono text-xs break-all text-muted-foreground">{invite.instanceHost}</p>
  </Frame>
);

// Turned away before anything is written: the link stays unused for the person it was made for.
export const ExistingAccountScreen = ({ invite }: { invite: ProtoInvite }) => (
  <Frame>
    <MarkPair person={invite.person.name} />
    <p className={cn(microheaderClass, "mt-4")}>Invitation</p>
    <h1 className={cn(displayTitleClass, "mt-1.5 text-[1.5rem]")}>You already have an account here</h1>
    <p className="mt-2 text-sm text-muted-foreground">This invitation is for someone new.</p>
    <SignedInAs invite={invite} className="mt-6 justify-center" />
    <Button type="button" className="mt-4 w-full">
      Go to {invite.instanceName}
    </Button>
    <p className="mt-4 text-xs text-muted-foreground">The link stays unused, so it still works for the person it was meant for.</p>
  </Frame>
);
