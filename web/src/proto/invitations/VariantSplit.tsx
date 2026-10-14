import { Logo } from "@/components/Logo";
import { Microheader } from "@/components/Microheader";
import { displayTitleClass } from "@/components/PageHeader";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { ProtoGrant, ProtoInvite } from "@/proto/invitations/data";
import { ExpiryLine, LATER_LINE, ProjectsLine, ProviderButtons, SignedInAs } from "@/proto/invitations/parts";

// A definition list: the workspace as the term, each role as "Name — what it does" in running text.
const GrantTerm = ({ grant, signedIn }: { grant: ProtoGrant; signedIn: boolean }) => (
  <div className="space-y-2 py-4 first:pt-0 last:pb-0">
    <dt className="text-[15px] font-medium break-words">{grant.workspace}</dt>
    {grant.roles.map((role) => (
      <dd key={role.id} className="text-[13px] leading-5 text-muted-foreground">
        <span className="font-medium text-foreground">{role.name}</span>
        {role.description && <span> — {role.description}</span>}
      </dd>
    ))}
    {signedIn && (
      <dd>
        <ProjectsLine projects={grant.projects} />
      </dd>
    )}
  </div>
);

export const VariantSplit = ({ invite, signedIn }: { invite: ProtoInvite; signedIn: boolean }) => (
  <div className="@container flex min-h-full items-center justify-center px-4 py-10 sm:px-8 sm:py-12">
    <div className="grid w-full max-w-5xl items-center gap-8 @3xl:grid-cols-[minmax(0,1fr)_minmax(0,26rem)] @3xl:gap-14">
      <div className="space-y-6">
        <Logo className="size-10 rounded-[10px] [&_svg]:size-6" />
        <div className="space-y-3">
          <h1 className={cn(displayTitleClass, "text-[2.25rem] @3xl:text-[2.75rem]")}>Join {invite.instanceName}</h1>
          <p className="font-mono text-xs break-all text-muted-foreground">{invite.instanceHost}</p>
        </div>
        {!signedIn && (
          <div className="max-w-sm space-y-3">
            <ProviderButtons />
          </div>
        )}
        {signedIn && (
          <div className="max-w-sm space-y-3">
            <SignedInAs invite={invite} />
            <div className="flex gap-2">
              <Button type="button">Accept invitation</Button>
              <Button type="button" variant="outline">
                Decline
              </Button>
            </div>
          </div>
        )}
        <ExpiryLine invite={invite} />
      </div>
      <div className="panel p-6 [--panel-opacity:72%] sm:p-7">
        <Microheader className="mb-4">What you get</Microheader>
        <dl className="divide-y divide-border">
          {invite.grants.map((grant) => (
            <GrantTerm key={grant.workspace} grant={grant} signedIn={signedIn} />
          ))}
        </dl>
        {signedIn && <p className="mt-4 border-t pt-3 text-xs text-muted-foreground">{LATER_LINE}</p>}
      </div>
    </div>
  </div>
);
